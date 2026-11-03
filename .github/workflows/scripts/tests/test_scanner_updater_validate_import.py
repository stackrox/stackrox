#!/usr/bin/env python3

import json
import os
import stat
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[4]
HELPER = (
    REPO_ROOT
    / ".github"
    / "workflows"
    / "scripts"
    / "scanner-updater-validate-import.py"
)
WORKFLOW = REPO_ROOT / ".github" / "workflows" / "scanner-updater-bundle.yaml"

VULN_REF = "11111111-1111-4111-8111-111111111111"
VULN_REF_2 = "22222222-2222-4222-8222-222222222222"
ENRICH_REF = "33333333-3333-4333-8333-333333333333"
VALID_TIMESTAMP = "2026-10-31T12:34:56"

FAKE_PSQL = r'''#!/usr/bin/env python3
import json
import os
import sys


with open(os.environ["FAKE_PSQL_FIXTURE"], encoding="utf-8") as fixture_file:
    fixture = json.load(fixture_file)

variables = {}
for argument in sys.argv[1:]:
    if argument.startswith("--set="):
        name, value = argument[len("--set="):].split("=", 1)
        variables[name] = value

query = sys.stdin.read()
with open(os.environ["FAKE_PSQL_CALLS"], "a", encoding="utf-8") as calls_file:
    calls_file.write(json.dumps({"argv": sys.argv[1:], "query": query}) + "\n")

if "metadata_sources" in variables:
    stage = "metadata"
elif "completed_operations" in variables:
    stage = "operation"
else:
    print("fake psql received no recognized query variable", file=sys.stderr)
    sys.exit(3)

if fixture.get("fail_stage") == stage:
    print(f"simulated {stage} SQL failure", file=sys.stderr)
    sys.exit(2)

raw_key = f"{stage}_raw"
if raw_key in fixture:
    sys.stdout.write(fixture[raw_key])
    sys.exit(0)

results_key = f"{stage}_results"
if results_key in fixture:
    result = fixture[results_key]
elif stage == "metadata":
    sources = json.loads(variables["metadata_sources"])
    metadata = fixture.get("metadata", {})
    result = [
        {"source": source, "timestamps": metadata.get(source, [])}
        for source in sources
    ]
else:
    operations = json.loads(variables["completed_operations"])
    configured = fixture.get("operations", {})
    result = []
    for operation in operations:
        state = configured.get(operation["ref"])
        if state is None:
            state = {"operation_kinds": [], "links": 0}
        result.append({
            "ref": operation["ref"],
            "operation_kinds": state["operation_kinds"],
            "links": state["links"],
        })

print(json.dumps(result, separators=(",", ":")))
'''


def completion(source, kind, ref, message_field="msg", count=999999):
    return json.dumps({
        message_field: "update imported",
        "bundle": f"bundles/{source}.json.zst",
        "kind": kind,
        "ref": ref,
        # This intentionally does not drive persistence validation.
        "count": count,
    })


def operation(kind, links):
    return {"operation_kinds": [kind], "links": links}


class ImportValidatorTest(unittest.TestCase):
    def setUp(self):
        self.temp_dir = tempfile.TemporaryDirectory()
        self.temp_path = Path(self.temp_dir.name)
        self.fake_psql = self.temp_path / "psql"
        self.fake_psql.write_text(FAKE_PSQL, encoding="utf-8")
        self.fake_psql.chmod(stat.S_IRUSR | stat.S_IWUSR | stat.S_IXUSR)

    def tearDown(self):
        self.temp_dir.cleanup()

    def run_case(self, sources, log_lines, fixture):
        log_path = self.temp_path / "updater.log"
        fixture_path = self.temp_path / "fixture.json"
        calls_path = self.temp_path / "calls.jsonl"
        log_path.write_text("\n".join(log_lines) + "\n", encoding="utf-8")
        fixture_path.write_text(json.dumps(fixture), encoding="utf-8")

        env = os.environ.copy()
        env["PATH"] = f"{self.temp_path}{os.pathsep}{env['PATH']}"
        env["FAKE_PSQL_FIXTURE"] = str(fixture_path)
        env["FAKE_PSQL_CALLS"] = str(calls_path)
        env["PYTHONDONTWRITEBYTECODE"] = "1"

        result = subprocess.run(
            [
                sys.executable,
                str(HELPER),
                "--log",
                str(log_path),
                "--db-conn",
                "host=fake dbname=fake",
                *sources,
            ],
            capture_output=True,
            text=True,
            check=False,
            env=env,
        )

        calls = []
        if calls_path.exists():
            calls = [json.loads(line) for line in calls_path.read_text().splitlines()]
        return result, calls

    @staticmethod
    def healthy_fixture(sources, operations):
        return {
            "metadata": {source: [VALID_TIMESTAMP] for source in sources},
            "operations": operations,
        }

    def run_metadata_case(self, timestamps):
        fixture = self.healthy_fixture(
            ["manual"], {VULN_REF: operation("vulnerability", 4)}
        )
        fixture["metadata"]["manual"] = timestamps
        return self.run_case(
            ["manual"],
            [completion("manual", "vulnerability", VULN_REF)],
            fixture,
        )[0]

    def test_accepts_vulnerability_and_enrichment_completion_formats(self):
        sources = ["manual", "epss"]
        fixture = self.healthy_fixture(
            sources,
            {
                VULN_REF: operation("vulnerability", 24),
                ENRICH_REF: operation("enrichment", 400000),
            },
        )
        result, _ = self.run_case(
            sources,
            [
                completion(
                    "manual", "vulnerability", VULN_REF, message_field="message"
                ),
                completion("epss", "enrichment", ENRICH_REF),
            ],
            fixture,
        )

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("source 'manual': imported", result.stdout)
        self.assertIn("effective_links=24", result.stdout)
        self.assertIn("source 'epss': imported", result.stdout)
        self.assertIn("effective_links=400000", result.stdout)

    def test_sums_multiple_operations_for_one_bundle(self):
        fixture = self.healthy_fixture(
            ["manual"],
            {
                VULN_REF: operation("vulnerability", 0),
                VULN_REF_2: operation("vulnerability", 7),
            },
        )
        result, _ = self.run_case(
            ["manual"],
            [
                completion("manual", "vulnerability", VULN_REF),
                completion("manual", "vulnerability", VULN_REF_2),
            ],
            fixture,
        )

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("effective_links=7", result.stdout)
        self.assertIn("operations=2", result.stdout)

    def test_accepts_semantically_valid_timestamp(self):
        result = self.run_metadata_case(["2026-10-31 12:34:56.123456"])

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_rejects_missing_metadata_row(self):
        result = self.run_metadata_case([])

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("manual: missing row in last_vuln_update", result.stdout)

    def test_rejects_null_metadata_timestamp(self):
        result = self.run_metadata_case([None])

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("manual: NULL update_timestamp", result.stdout)

    def test_rejects_go_zero_metadata_timestamp(self):
        result = self.run_metadata_case(["0001-01-01T00:00:00"])

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("not later than Unix epoch", result.stdout)

    def test_rejects_epoch_metadata_timestamp(self):
        result = self.run_metadata_case(["1970-01-01T00:00:00"])

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("not later than Unix epoch", result.stdout)

    def test_rejects_malformed_metadata_timestamp(self):
        result = self.run_metadata_case(["31 October 2026"])

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("malformed update_timestamp", result.stdout)

    def test_rejects_ambiguous_metadata_rows(self):
        result = self.run_metadata_case([VALID_TIMESTAMP, VALID_TIMESTAMP])

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("ambiguous last_vuln_update result", result.stdout)

    def test_rejects_missing_completion_event(self):
        fixture = {"metadata": {"manual": [VALID_TIMESTAMP]}}
        result, _ = self.run_case(
            ["manual"],
            [json.dumps({"msg": "starting bundle update"})],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("manual: missing update imported completion event", result.stdout)

    def test_rejects_unknown_completion_kind(self):
        fixture = {"metadata": {"manual": [VALID_TIMESTAMP]}}
        result, _ = self.run_case(
            ["manual"],
            [completion("manual", "unknown", VULN_REF)],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unknown kind 'unknown'", result.stdout)

    def test_rejects_missing_database_operation_ref(self):
        fixture = self.healthy_fixture(["manual"], {})
        result, _ = self.run_case(
            ["manual"],
            [completion("manual", "vulnerability", VULN_REF)],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("is missing from the database", result.stdout)

    def test_rejects_zero_vulnerability_links_even_with_positive_log_count(self):
        fixture = self.healthy_fixture(
            ["manual"], {VULN_REF: operation("vulnerability", 0)}
        )
        result, _ = self.run_case(
            ["manual"],
            [completion("manual", "vulnerability", VULN_REF, count=1000000)],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("zero effective vulnerability relations", result.stdout)

    def test_rejects_zero_enrichment_links(self):
        fixture = self.healthy_fixture(
            ["epss"], {ENRICH_REF: operation("enrichment", 0)}
        )
        result, _ = self.run_case(
            ["epss"],
            [completion("epss", "enrichment", ENRICH_REF)],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("zero effective enrichment relations", result.stdout)

    def test_rejects_malformed_relevant_log_record(self):
        fixture = {"metadata": {"manual": [VALID_TIMESTAMP]}}
        result, _ = self.run_case(
            ["manual"],
            ['{"msg":"update imported","bundle":"bundles/manual.json.zst"'],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("malformed JSON", result.stdout)

    def test_rejects_metadata_sql_failure(self):
        fixture = self.healthy_fixture(
            ["manual"], {VULN_REF: operation("vulnerability", 4)}
        )
        fixture["fail_stage"] = "metadata"
        result, _ = self.run_case(
            ["manual"],
            [completion("manual", "vulnerability", VULN_REF)],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("database metadata query failed with exit status 2", result.stdout)

    def test_rejects_operation_sql_failure(self):
        fixture = self.healthy_fixture(
            ["manual"], {VULN_REF: operation("vulnerability", 4)}
        )
        fixture["fail_stage"] = "operation"
        result, _ = self.run_case(
            ["manual"],
            [completion("manual", "vulnerability", VULN_REF)],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("database operation query failed with exit status 2", result.stdout)

    def test_rejects_duplicate_completion_ref(self):
        event = completion("manual", "vulnerability", VULN_REF)
        fixture = self.healthy_fixture(
            ["manual"], {VULN_REF: operation("vulnerability", 4)}
        )
        result, _ = self.run_case(["manual"], [event, event], fixture)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate completion event for ref", result.stdout)

    def test_rejects_duplicate_json_keys_in_completion(self):
        fixture = {"metadata": {"manual": [VALID_TIMESTAMP]}}
        event = (
            '{"msg":"update imported","bundle":"bundles/manual.json.zst",'
            '"kind":"vulnerability","kind":"enrichment",'
            f'"ref":"{VULN_REF}"}}'
        )
        result, _ = self.run_case(["manual"], [event], fixture)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate JSON object key 'kind'", result.stdout)

    def test_rejects_conflicting_completion_ref(self):
        fixture = self.healthy_fixture(
            ["manual", "epss"],
            {VULN_REF: operation("vulnerability", 4)},
        )
        result, _ = self.run_case(
            ["manual", "epss"],
            [
                completion("manual", "vulnerability", VULN_REF),
                completion("epss", "enrichment", VULN_REF),
            ],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("conflicting completion events for ref", result.stdout)

    def test_rejects_database_kind_mismatch(self):
        fixture = self.healthy_fixture(
            ["manual"], {VULN_REF: operation("enrichment", 4)}
        )
        result, _ = self.run_case(
            ["manual"],
            [completion("manual", "vulnerability", VULN_REF)],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("database kind 'enrichment'", result.stdout)

    def test_rejects_ambiguous_psql_output(self):
        fixture = self.healthy_fixture(
            ["manual"], {VULN_REF: operation("vulnerability", 4)}
        )
        fixture["metadata_raw"] = "[]\n[]\n"
        result, _ = self.run_case(
            ["manual"],
            [completion("manual", "vulnerability", VULN_REF)],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("metadata query returned ambiguous output", result.stdout)

    def test_rejects_unsafe_source_name_before_querying(self):
        fixture = {"metadata": {}}
        result, calls = self.run_case(
            ["manual';DROP_TABLE"],
            [],
            fixture,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("invalid expected source name", result.stdout)
        self.assertEqual(calls, [])

    def test_passes_sources_and_refs_as_psql_variables(self):
        fixture = self.healthy_fixture(
            ["manual"], {VULN_REF: operation("vulnerability", 4)}
        )
        result, calls = self.run_case(
            ["manual"],
            [completion("manual", "vulnerability", VULN_REF)],
            fixture,
        )

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(len(calls), 2)
        all_query_text = "\n".join(call["query"] for call in calls)
        self.assertNotIn("manual", all_query_text)
        self.assertNotIn(VULN_REF, all_query_text)
        self.assertIn(":'metadata_sources'", all_query_text)
        self.assertIn(":'completed_operations'", all_query_text)
        self.assertTrue(
            any("manual" in argument for argument in calls[0]["argv"])
        )
        self.assertTrue(
            any(VULN_REF in argument for argument in calls[1]["argv"])
        )

    def test_disposable_workflow_is_guarded_and_routes_corrected_cases(self):
        workflow = WORKFLOW.read_text(encoding="utf-8")
        self.assertIn(
            "if: github.repository == 'stackrox/stackrox' && github.event_name == 'workflow_dispatch'",
            workflow,
        )
        self.assertIn('[[ "$TEST_REF" == "refs/heads/alkmim/ROX-36961-t1-negative-tests" ]]', workflow)
        self.assertIn('[[ "$DRY_RUN" == "true" && "$FORCE" == "true" ]]', workflow)
        self.assertIn("        - N3c\n", workflow)
        self.assertIn("        - N4-go-zero\n", workflow)
        self.assertIn("max-parallel: 1", workflow)
        self.assertIn("scanner-updater-bundle-t1-${{ github.ref }}", workflow)

        assemble = workflow.index("    - name: Assemble bundle\n")
        n3c_injection = workflow.index("    - name: Inject N3c unsupported manual record\n")
        zip_validation = workflow.index("    - name: Validate ZIP entries\n")
        self.assertLess(assemble, n3c_injection)
        self.assertLess(n3c_injection, zip_validation)
        self.assertIn("run: python3 /tmp/t1-tests/inject.py N3c", workflow)

        for secret in (
            "secrets.GOOGLE_SA_CIRCLECI_SCANNER",
            "gs://scanner-v4-test/${GCS_PREFIX}/test-runs/${GITHUB_RUN_ID}/${GITHUB_RUN_ATTEMPT}/${STREAM}/vulnerabilities.zip",
        ):
            self.assertIn(secret, workflow)
        self.assertNotIn("GOOGLE_SA_STACKROX_HUB_VULN_DUMP_UPLOADER", workflow)

        prepare = (REPO_ROOT / ".github/workflows/scripts/t1-tests/prepare-run.py").read_text()
        importer = (REPO_ROOT / ".github/workflows/scripts/t1-tests/run-import.py").read_text()
        injector = (REPO_ROOT / ".github/workflows/scripts/t1-tests/inject.py").read_text()
        verifier = (REPO_ROOT / ".github/workflows/scripts/t1-tests/verify-object.py").read_text()
        self.assertIn("CORRECTED_CASES = {\"healthy\", \"N3c\", \"N4-go-zero\"}", prepare)
        self.assertIn("scanner-updater-validate-import.py", prepare)
        self.assertIn("lib.extract(repo, dest)", prepare)
        self.assertIn("if case in CORRECTED_CASES", prepare)
        self.assertIn("configured_sources = lib.sources(repo, stream", prepare)
        self.assertIn("scanner-updater-validate-import.py", importer)
        self.assertIn('run_manifest.get("configured_sources")', importer)
        self.assertIn("TIMESTAMPTZ '0001-01-01 00:00:00+00'", importer)
        self.assertIn('"N3c"', injector)
        self.assertIn("manual_record(zero=True)", injector)
        self.assertIn("zero effective vulnerability relations", verifier)
        self.assertIn("not later than Unix epoch", verifier)


if __name__ == "__main__":
    unittest.main()
