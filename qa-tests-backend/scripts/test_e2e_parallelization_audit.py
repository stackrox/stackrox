import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from e2e_parallelization_audit import analyze, parse_events


def event(execution, specification, event_type, timestamp, **attributes):
    return {
        "schema_version": 1,
        "run_id": "run-1",
        "lane_id": "qa-part-1",
        "execution_id": execution,
        "specification": specification,
        "event_type": event_type,
        "timestamp": timestamp,
        "attributes": attributes,
    }


class ParallelizationAuditTest(unittest.TestCase):
    def test_parses_sentinel_embedded_in_actions_log_line(self):
        line = (
            "2026-10-01T12:00:00Z 2026/10/01 12:00:00 "
            'e2e_parallel_audit {"schema_version":1,"event_type":"spec_start"}\n'
        )
        events, malformed = parse_events([line, "ordinary output\n"])
        self.assertEqual(1, len(events))
        self.assertEqual(0, malformed)

    def test_reports_overlapping_kubernetes_read_write_on_same_resource(self):
        events = [
            event("a", "SpecA", "spec_start", "2026-10-01T12:00:00Z"),
            event("a", "SpecA", "operation", "2026-10-01T12:00:01Z", system="kubernetes", method="GET", path="/api/v1/namespaces/shared/configmaps/cm"),
            event("b", "SpecB", "spec_start", "2026-10-01T12:00:02Z"),
            event("b", "SpecB", "operation", "2026-10-01T12:00:03Z", system="kubernetes", method="PATCH", path="/api/v1/namespaces/shared/configmaps/cm"),
            event("a", "SpecA", "spec_end", "2026-10-01T12:00:05Z"),
            event("b", "SpecB", "spec_end", "2026-10-01T12:00:06Z"),
        ]
        report = analyze(events)
        self.assertEqual(1, len(report["shared_write_candidates"]))
        self.assertEqual("kubernetes", report["shared_write_candidates"][0]["system"])
        self.assertEqual("overlapping", report["shared_write_candidates"][0]["temporal_relation"])

    def test_does_not_flag_read_read_on_sequential_specs(self):
        events = [
            event("a", "SpecA", "spec_start", "2026-10-01T12:00:00Z"),
            event("a", "SpecA", "operation", "2026-10-01T12:00:01Z", system="kubernetes", method="GET", path="/api/v1/namespaces/shared/pods"),
            event("a", "SpecA", "spec_end", "2026-10-01T12:00:02Z"),
            event("b", "SpecB", "spec_start", "2026-10-01T12:00:02Z"),
            event("b", "SpecB", "operation", "2026-10-01T12:00:03Z", system="kubernetes", method="GET", path="/api/v1/namespaces/shared/pods"),
            event("b", "SpecB", "spec_end", "2026-10-01T12:00:04Z"),
        ]
        report = analyze(events)
        self.assertEqual(0, len(report["shared_write_candidates"]))
        self.assertEqual(0, report["overlapping_specification_pairs"])

    def test_flags_shared_writes_between_currently_sequential_specs_as_candidate(self):
        events = [
            event("a", "SpecA", "spec_start", "2026-10-01T12:00:00Z"),
            event("a", "SpecA", "operation", "2026-10-01T12:00:01Z", system="kubernetes", method="POST", path="/api/v1/namespaces/shared/configmaps/cm"),
            event("a", "SpecA", "spec_end", "2026-10-01T12:00:02Z"),
            event("b", "SpecB", "spec_start", "2026-10-01T12:00:03Z"),
            event("b", "SpecB", "operation", "2026-10-01T12:00:04Z", system="kubernetes", method="GET", path="/api/v1/namespaces/shared/configmaps/cm"),
            event("b", "SpecB", "spec_end", "2026-10-01T12:00:05Z"),
        ]
        report = analyze(events)
        self.assertEqual(1, len(report["shared_write_candidates"]))
        self.assertEqual("sequential", report["shared_write_candidates"][0]["temporal_relation"])

    def test_reports_coarse_central_service_and_global_auth_conflicts(self):
        events = [
            event("a", "SpecA", "spec_start", "2026-10-01T12:00:00Z"),
            event("a", "SpecA", "operation", "2026-10-01T12:00:01Z", system="central-grpc", service="stackrox.api.v1.RoleService", method="GetRole"),
            event("a", "SpecA", "operation", "2026-10-01T12:00:01Z", system="process-global", key="central-auth-configuration", access="write"),
            event("b", "SpecB", "spec_start", "2026-10-01T12:00:02Z"),
            event("b", "SpecB", "operation", "2026-10-01T12:00:03Z", system="central-grpc", service="stackrox.api.v1.RoleService", method="DeleteRole"),
            event("b", "SpecB", "operation", "2026-10-01T12:00:04Z", system="process-global", key="central-auth-configuration", access="write"),
            event("a", "SpecA", "spec_end", "2026-10-01T12:00:05Z"),
            event("b", "SpecB", "spec_end", "2026-10-01T12:00:06Z"),
        ]
        report = analyze(events)
        self.assertEqual(2, len(report["shared_write_candidates"]))
        self.assertEqual(
            {"central-grpc", "process-global"},
            {finding["system"] for finding in report["shared_write_candidates"]},
        )


if __name__ == "__main__":
    unittest.main()
