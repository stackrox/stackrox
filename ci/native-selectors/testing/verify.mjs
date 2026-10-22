import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {createHash} from 'node:crypto';
import {execFileSync} from 'node:child_process';
import {Lexer, Parser, Evaluator, data} from '@actions/expressions';
import YAML from 'yaml';

const root = path.resolve('../..');
const read = file => fs.readFileSync(path.join(root, file), 'utf8');
const workflow = name => YAML.parse(read(`.github/workflows/${name}`));
const files = ['unit-tests.yaml', 'style.yaml', 'check-crd-compatibility.yaml',
  'scanner-db-integration-tests.yaml', 'build.yaml', 'native-ci-tests.yaml', 'e2e-dispatch.yaml'];
const workflows = Object.fromEntries(files.map(file => [file, workflow(file)]));
const units = workflows['unit-tests.yaml'];
const e2e = workflows['e2e-dispatch.yaml'];
const detector = workflow('detect-changes.yml').jobs.detect.steps[0];
const report = YAML.parse(read('.github/actions/ci-selection-summary/action.yml'));
const original = JSON.parse(read('ci/native-selectors/testing/base-workflows.json'));
const hash = text => createHash('sha256').update(text).digest('hex');
const hashJSON = value => hash(JSON.stringify(value));
const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'native-job-local-'));
const binary = path.resolve('.test-build/file-facts');
const unknown = JSON.stringify({version: 1, complete: false, any: [], every: []});
const statuses = ['always', 'cancelled', 'success', 'failure'].map(name => ({
  name, minArgs: 0, maxArgs: 0, call: () => new data.BooleanData(name === 'always' || name === 'success'),
}));
function evaluate(source, context) {
  if (typeof source !== 'string') return String(source);
  const expression = source.trim().replace(/^\$\{\{\s*|\s*\}\}$/g, '');
  const ast = new Parser(new Lexer(expression).lex().tokens,
    ['github', 'inputs', 'needs', 'steps', 'matrix', 'env'], statuses).parse();
  return new Evaluator(ast, JSON.parse(JSON.stringify(context), data.reviver),
    new Map(statuses.map(f => [f.name, f]))).evaluate().coerceString();
}
function classify(files, options = {}) {
  if (options.unknown) return unknown;
  return execFileSync(binary, [], {cwd: root, encoding: 'utf8', input: JSON.stringify({
    base: 'fixture-base', head: 'fixture-head', files,
  })}).trim();
}
function collect(files, options = {}) {
  const output = path.join(temp, 'detected');
  const raw = path.join(temp, 'native-ci-files.json');
  fs.writeFileSync(output, '');
  fs.rmSync(raw, {force: true});
  const mock = `gh() {
    [[ "$FIXTURE_FAILURE" != 'true' ]] || return 1
    if [[ "$2" == */files ]]; then printf '%s' "$FIXTURE_PAGES";
    else printf '%s\\t%s' "$FIXTURE_HEAD" "$FIXTURE_BASE"; fi
  }\n`;
  execFileSync(process.env.NATIVE_TEST_BASH || 'bash', ['-c', mock + detector.run], {env: {
    ...process.env, GH_REPO: 'fixture/repo', GH_TOKEN: '', PR_NUMBER: options.nonPR ? '' : '1',
    BASE_SHA: 'fixture-base', EXPECTED_HEAD: 'fixture-head', RUNNER_TEMP: temp,
    FIXTURE_HEAD: options.stale ? 'new-head' : 'fixture-head',
    FIXTURE_BASE: options.staleBase ? 'new-base' : 'fixture-base',
    FIXTURE_FAILURE: String(!!options.failure), FIXTURE_PAGES: options.pages || JSON.stringify([files.map(filename => ({filename}))]),
    RUN_ALL: units.jobs['detect-changes'].with.run_all,
    WORKFLOW_FILE_PATH: detector.env.WORKFLOW_FILE_PATH, GITHUB_OUTPUT: output,
  }, stdio: ['ignore', 'pipe', 'pipe']});
  const text = fs.readFileSync(output, 'utf8');
  return {files: text.match(/files<<EOF\n([\s\S]*?)\nEOF(?:\n|$)/)?.[1] || '',
    ui_only: text.includes('ui_only=true\n') ? 'true' : '',
    facts: fs.existsSync(raw) && !options.unknown ? execFileSync(binary, [], {
      cwd: root, encoding: 'utf8', input: fs.readFileSync(raw),
    }).trim() : unknown};
}
function context(paths, labels = [], options = {}) {
  const currentLabels = options.liveLabels || labels;
  const ctx = {
    github: {event_name: options.event || 'pull_request', ref: options.ref || 'refs/pull/1/merge', event: {pull_request: {
      labels: labels.map(name => ({name})), draft: !!options.draft, head: {repo: {fork: !!options.fork}},
    }}},
    needs: {'detect-changes': {result: 'success', outputs: collect(paths, {
      ...options, nonPR: options.event && options.event !== 'pull_request',
    })}},
    steps: {labels: {outcome: options.labelFailure ? 'failure' : 'success', outputs: {
      labels: options.labelFailure ? '' : JSON.stringify(currentLabels),
    }}}, matrix: {pg: '16'},
  };
  ctx.needs['should-dispatch'] = {result: 'success', outputs: Object.fromEntries(
    Object.entries(e2e.jobs['should-dispatch'].outputs).map(([key, value]) => [key, evaluate(value, ctx)]))};
  return ctx;
}
const local = new Map();
for (const [file, doc] of Object.entries(workflows)) for (const [name, job] of Object.entries(doc.jobs)) {
  const step = job.steps?.find(step => step.id === 'selection');
  if (step) local.set(name === 'define-job-matrix' ? 'build-images' : name, {file, name, job, step});
}
const consumers = Object.keys(e2e.jobs).filter(name => e2e.jobs[name].uses?.includes('/e2e-'));
function explain(target, ctx) {
  const {job, step} = local.get(target);
  const env = {...ctx.env};
  if (job.env?.CI_FILE_FACTS) env.CI_FILE_FACTS = evaluate(job.env.CI_FILE_FACTS, ctx);
  const inputs = Object.fromEntries(Object.entries(report.inputs).map(([key, value]) => [key, value.default || '']));
  for (const [key, value] of Object.entries(step.with)) inputs[key] = key === 'note' ? value : evaluate(value, {...ctx, env});
  return {candidate: inputs.candidate === 'true', enforced: inputs.enforced === 'true',
    run: evaluate(report.runs.steps[0].env.RUN, {inputs}) === 'true'};
}
function candidate(target, ctx) {
  if (local.has(target)) return explain(target, ctx).candidate;
  // Remove execution restrictions to observe the candidate inside the actual consumer expression.
  const unrestricted = structuredClone(ctx);
  unrestricted.github.event_name = 'pull_request';
  unrestricted.github.event.pull_request.head.repo.fork = false;
  unrestricted.needs['should-dispatch'].outputs.enforced = 'true';
  const labels = JSON.parse(unrestricted.needs['should-dispatch'].outputs.labels);
  unrestricted.needs['should-dispatch'].outputs.labels = JSON.stringify([...labels, 'konflux-build']);
  return evaluate(e2e.jobs[target].with['should-run'], unrestricted) === 'true';
}
function dispatch(ctx) {
  return {jobs: Object.fromEntries(consumers.map(name => [name, evaluate(e2e.jobs[name].with['should-run'], ctx) === 'true'])),
    'wait-for-gha-images': String(!ctx.github.event.pull_request?.head.repo.fork && explain('wait-for-images', ctx).run),
    'wait-for-konflux-images': String(!ctx.github.event.pull_request?.head.repo.fork && explain('wait-for-konflux-images', ctx).run)};
}

try {
  const comparison = JSON.parse(read('ci/native-selectors/testing/comparison.json'));
  assert.equal(comparison.sha, '34873a53e3b60b5fc7b50b1446b02d11a1eb0fca');
  for (const [file, digest] of Object.entries(comparison.fixtures)) {
    assert.equal(hash(read(`ci/native-selectors/testing/fixtures/${file}`)), digest, `${file}: pinned evidence changed`);
  }
  assert.equal(local.size + consumers.length, 29, 'account for every Actions target in the pinned comparison');
  const ui = collect(['ui/App.tsx']);
  assert.equal(ui.files, '^ui/App.tsx$');
  assert.equal(ui.ui_only, 'true');
  assert(JSON.parse(ui.facts).every.includes('ui'));
  assert.equal(collect(['ui/App.tsx', 'README.md']).ui_only, '');
  assert(collect([], {pages: '[[{"filename":"ui/App.tsx"}],[{"filename":"sensor/old.go","status":"removed"}]]'}).files.includes('^sensor/old.go$'));
  const rename = collect([], {pages: '[[{"filename":"ui/new.tsx","previous_filename":"sensor/kubernetes/client/old.go"}]]'});
  assert(rename.files.includes('^sensor/kubernetes/client/old.go$'));
  assert(!JSON.parse(rename.facts).every.includes('ui'));
  const many = Array.from({length: 1000}, (_, i) => `ui/${i}/${'x'.repeat(90)}.tsx`);
  assert.equal(collect(['central/main.go', ...many]).ui_only, '');
  const overridden = collect(['Makefile', ...many]);
  assert.equal(overridden.files, '');
  assert.equal(JSON.parse(overridden.facts).complete, true, 'legacy run_all does not erase classified facts');
  assert.equal(JSON.parse(collect([]).facts).complete, true, 'known empty differs from a failed collection');
  for (const [paths, options] of [
    [[], {pages: '[{"error":true}]'}], [[''], {}], [['README.md'], {failure: true}], [['README.md'], {stale: true}], [['README.md'], {staleBase: true}],
    [['README.md'], {nonPR: true}], [['ui/a\nb.tsx'], {}], [['ui/a\rb.tsx'], {}],
    [Array.from({length: 3000}, (_, i) => `ui/${i}.tsx`), {}], [['ui/' + 'a'.repeat(200001)], {}],
  ]) {
    const result = collect(paths, options);
    assert.equal(result.files, '');
    assert.equal(result.ui_only, '');
    assert.equal(result.facts, unknown);
  }
  const groups = JSON.parse(read('ci/native-selectors/file-groups.json'));
  assert.equal(Object.keys(groups).length, 19);
  for (const paths of [[], ['ui/a.tsx'], ['ui/a.tsx', 'OWNERS'], ['scanner/config.yaml'],
    ['scanner/config.yaml', 'README.md'], ['central/sac/store.go', 'sensor/kubernetes/client/client.go'],
    ['pkg/concurrency/context.go'], ['elsewhere/schema.proto'], ['elsewhere/foo.go.txt'], ['line\nbreak.go']]) {
    const result = JSON.parse(classify(paths));
    for (const [group, patterns] of Object.entries(groups)) {
      const matches = file => patterns.some(pattern => new RegExp(pattern).test(file));
      assert.equal(result.any.includes(group), paths.some(matches), `any/${group}`);
      assert.equal(result.every.includes(group), paths.length > 0 && paths.every(matches), `every/${group}`);
    }
  }
  for (const input of ['null', '{}', '{"base":"a","head":"b","files":null}', '{"base":"a","head":"b","files":[""]}', '{"base":"a","head":"b","files":[3]}',
    '{"base":"a","head":"b","files":[]} {}']) {
    assert.throws(() => execFileSync(binary, [], {cwd: root, input, stdio: ['pipe', 'pipe', 'pipe']}));
  }
  const invalidGroups = path.join(temp, 'invalid-groups.json');
  for (const groups of ['{"invalid":["["]}', 'null', '{}', '{"empty":[]}', '{"":["^ui/"]}']) {
    fs.writeFileSync(invalidGroups, groups);
    assert.throws(() => execFileSync(binary, ['--groups', invalidGroups], {cwd: root,
      input: '{"base":"a","head":"b","files":[]}', stdio: ['pipe', 'pipe', 'pipe']}));
  }
  console.log('Collector pagination, renames, legacy overrides, empty/unknown diffs and limits checked; 19 any/every path groups exercised.');

  const baseline = JSON.parse(read('ci/native-selectors/testing/fixtures/expected.json'));
  const scenarios = JSON.parse(read('ci/native-selectors/testing/fixtures/scenarios.json'));
  let compared = 0;
  for (const scenario of scenarios) {
    const labels = [...scenario.labels, ...(scenario.enforced ? ['ci-dispatcher-enforce'] : [])];
    const ctx = context(scenario.files, labels, scenario);
    if (scenario.event === 'pull_request') for (const target of [...local.keys(), ...consumers]) {
      assert.equal(candidate(target, ctx), baseline[scenario.name].selected.includes(target), `${scenario.name}/${target}`);
      compared++;
    }
    const actual = dispatch(ctx), expected = baseline[scenario.name].dispatch;
    assert.deepEqual(actual.jobs, JSON.parse(expected.jobs), `${scenario.name}/consumer conditions`);
    assert.equal(actual['wait-for-gha-images'], expected['wait-for-gha-images'], `${scenario.name}/ordinary waiter`);
    assert.equal(actual['wait-for-konflux-images'], expected['wait-for-konflux-images'], `${scenario.name}/Konflux waiter`);
    for (const [name, runs] of Object.entries(actual.jobs)) if (runs) {
      // A legacy explicit Konflux label can select its consumer without making an image build available.
      const waiter = name.includes('konflux') ? 'wait-for-konflux-images' : 'wait-for-gha-images';
      if (scenario.enforced) assert.equal(actual[waiter], 'true', `${scenario.name}/${name}: missing prerequisite`);
    }
  }
  console.log(`${compared} candidate comparisons across 29 Actions targets and ${scenarios.length} E2E dispatch/waiter scenarios match the unchanged pinned fixtures.`);

  let gates = 0;
  for (const [key, saved] of Object.entries(original.jobs)) {
    const split = key.indexOf('/'), file = key.slice(0, split), name = key.slice(split + 1);
    const current = structuredClone(workflows[file].jobs[name]);
    const steps = current.steps.filter(step => step.id !== 'selection');
    delete current.steps;
    const waiter = name.startsWith('wait-for-');
    if (name === 'define-job-matrix') {
      assert.equal(current.if, '${{ !cancelled() }}', 'image production must not depend on successful file collection');
      delete current.if;
    }
    if (waiter) {
      assert.deepEqual(current.needs, ['detect-changes', 'should-dispatch']);
      assert.equal(evaluate(current.if, context([], [], {fork: true})), 'false');
      current.needs = saved.needs;
      current.if = saved.if;
      delete current.outputs.run;
    }
    if (current.env) {delete current.env.CI_FILE_FACTS; if (!Object.keys(current.env).length) delete current.env;}
    if (saved.needs === undefined) delete current.needs;
    assert.equal(hashJSON(current), saved.structure, `${key}: matrix, check name, outer guard or execution environment changed`);
    assert.equal(hashJSON(steps.map(({if: condition, ...step}) => step)), saved.steps, `${key}: execution commands changed`);
    const ctx = context(['README.md']);
    for (const run of ['false', 'true', '']) {
      ctx.steps.selection = {outputs: {run}};
      for (const [i, step] of steps.entries()) {
        const gated = waiter ? i === steps.length - 1 : i > 0 && !['style-check', 'define-job-matrix'].includes(name);
        const expected = gated && run === 'false' ? 'false' : evaluate(saved.conditions[i], ctx);
        assert.equal(evaluate(step.if || 'success()', ctx), expected, `${key}/${step.name || step.uses}/${run}`);
        gates++;
      }
    }
    if (file !== 'build.yaml') assert(workflows[file].on.pull_request.types.includes('labeled') && workflows[file].on.pull_request.types.includes('unlabeled'));
  }
  for (const [file, digest] of Object.entries(original.headers)) {
    const {jobs, ...header} = structuredClone(workflows[file]);
    header.on.pull_request.types = header.on.pull_request.types.filter(type => !['labeled', 'unlabeled'].includes(type));
    assert.equal(hashJSON(header), digest, `${file}: existing non-PR triggers or permissions changed`);
    assert(!header.on.pull_request.paths && !header.on.pull_request['paths-ignore']);
  }
  for (const [key, digest] of Object.entries(original.unchangedJobs)) {
    const split = key.indexOf('/');
    assert.equal(hashJSON(workflows[key.slice(0, split)].jobs[key.slice(split + 1)]), digest, `${key}: unrelated execution changed`);
  }
  for (const [file, digest] of Object.entries(original.children)) assert.equal(hash(read(`.github/workflows/${file}`)), digest);
  const enforced = ['ci-dispatcher-enforce'];
  const unrelated = context(['README.md'], enforced);
  assert.equal(explain('go-postgres', unrelated).run, false);
  assert.equal(explain('go-postgres', context(['README.md'])).run, true, 'shadow keeps eligible work');
  assert.equal(explain('go-postgres', context(['README.md'], [...enforced, 'area/central'])).run, true);
  assert.equal(explain('go-postgres', context(['README.md'], ['area/central'])).run, true, 'removing enforcement restores shadow');
  const runAllUI = context(['ui/package.json'], [...enforced, 'ci-run-all-tests']);
  assert.equal(candidate('sensor-integration-tests', runAllUI), true);
  assert.equal(evaluate(units.jobs['sensor-integration-tests'].if, runAllUI), 'false', 'run-all cannot override an existing outer guard');
  const labelUI = context(['README.md'], [...enforced, 'area/ui']);
  assert.equal(candidate('ui', labelUI), true);
  assert.equal(evaluate(units.jobs.ui.if, labelUI), 'false', 'a label cannot bypass the existing UI job guard');
  assert.equal(dispatch(context(['README.md'], [], {liveLabels: [...enforced, 'e2e-byodb-tests']})).jobs['e2e-byodb-tests'], true);
  assert.equal(dispatch(context(['README.md'], [...enforced, 'e2e-byodb-tests'], {liveLabels: enforced})).jobs['e2e-byodb-tests'], false);
  const refreshFailure = context(['README.md'], [...enforced, 'ci-run-all-tests'], {labelFailure: true});
  assert.equal(refreshFailure.needs['should-dispatch'].outputs.enforced, 'false');
  assert.equal(dispatch(refreshFailure).jobs['e2e-byodb-tests'], false, 'failed refresh returns to legacy eligibility');
  const apiFailure = context(['README.md'], [...enforced, 'ci-run-all-tests'], {failure: true});
  assert.equal(dispatch(apiFailure).jobs['e2e-byodb-tests'], true);
  assert.equal(dispatch(apiFailure).jobs['e2e-qa-tests-gke'], false, 'unknown diff retains the QA skip default even with run-all');
  for (const event of ['push', 'workflow_dispatch']) {
    const nonPR = context([], [], {event});
    delete nonPR.github.event.pull_request;
    nonPR.steps.labels = {outcome: 'skipped', outputs: {}};
    nonPR.needs['should-dispatch'].outputs = Object.fromEntries(Object.entries(e2e.jobs['should-dispatch'].outputs)
      .map(([key, value]) => [key, evaluate(value, nonPR)]));
    const plan = dispatch(nonPR);
    assert(Object.values(plan.jobs).every(Boolean));
    assert.equal(plan['wait-for-gha-images'], 'true');
    assert.equal(plan['wait-for-konflux-images'], 'true');
  }
  const oldEvent = context(['README.md'], [...enforced, 'area/central'], {liveLabels: enforced});
  assert.equal(explain('go-postgres', oldEvent).candidate, true, 'ordinary reruns retain event labels');
  const missingReport = context(['go.mod'], [...enforced, 'konflux-build']);
  missingReport.steps.selection = {outputs: {}};
  assert.equal(evaluate(e2e.jobs['wait-for-images'].outputs.run, missingReport), 'true');
  assert.equal(evaluate(e2e.jobs['wait-for-konflux-images'].outputs.run, missingReport), 'true');
  for (const target of [...local.keys()]) {
    const decision = explain(target, context(['go.mod'], [...enforced, 'konflux-build']));
    assert(decision.candidate, `${target}: broad dependency change`);
  }
  const requirements = JSON.parse(read('ci/native-selectors/testing/fixtures/required-gha-checks.json'));
  let contexts = 0;
  for (const [file, jobs] of Object.entries(requirements)) for (const [name, checks] of Object.entries(jobs)) {
    assert(workflows[file].jobs[name], `${file}/${name}: missing required job`);
    contexts += checks.length;
  }
  for (const name of consumers) {
    const job = e2e.jobs[name], child = workflow(job.uses.split('/').at(-1));
    assert.equal(child.on.workflow_call.inputs['should-run'].type, 'boolean');
    assert.equal(evaluate(job.if, {needs: Object.fromEntries(job.needs.map(name => [name, {outputs: {run: 'false'}}]))}), 'false');
  }
  assert(!fs.existsSync(path.join(root, '.github/workflows/native-ci-selection.yaml')));
  assert(!fs.existsSync(path.join(root, 'ci/native-selectors/path_facts.py')));
  console.log(`${gates} preserved-step checks and ${contexts} required-context entries checked; label refresh, outer restrictions and missing image prerequisites exercised.`);

  // Execute the real matrix construction with local context functions, without building images.
  const matrixScript = workflows['build.yaml'].jobs['define-job-matrix'].steps.find(step => step.id === 'define-job-matrix').run;
  fs.mkdirSync(path.join(temp, 'scripts/ci'), {recursive: true});
  fs.writeFileSync(path.join(temp, 'scripts/ci/lib.sh'), `is_in_PR_context() { [[ "$FIXTURE_PR" == true ]]; }\nis_tagged() { [[ "$FIXTURE_TAG" == true ]]; }\npr_has_label() { [[ ",$FIXTURE_LABELS," == *",$1,"* ]]; }\n`);
  for (const [pr, tag, labels] of [[true, false, ''], [true, false, 'ci-build-all-arch,ci-build-prerelease,ci-build-race-condition-debug'], [false, false, ''], [false, true, '']]) {
    const output = path.join(temp, 'matrix'); fs.writeFileSync(output, '');
    execFileSync(process.env.NATIVE_TEST_BASH || 'bash', ['-c', matrixScript], {cwd: temp, env: {
      ...process.env, GITHUB_OUTPUT: output, FIXTURE_PR: String(pr), FIXTURE_TAG: String(tag), FIXTURE_LABELS: labels,
    }, stdio: ['ignore', 'pipe', 'pipe']});
    const matrix = JSON.parse(fs.readFileSync(output, 'utf8').trim().slice('matrix='.length));
    assert(matrix.build_and_push_main.arch.includes('amd64') && matrix.build_and_push_main.arch.includes('arm64'));
    assert(matrix.build_and_push_main.name.includes('STACKROX_BRANDING') && matrix.build_and_push_main.name.includes('RHACS_BRANDING'));
    if (!pr || labels) assert(matrix.build_and_push_main.arch.includes('s390x'));
  }
  console.log('Four real build-matrix constructions retain required image variants; image builds remain ungated.');

  const evidence = JSON.parse(read('ci/native-selectors/prow/source.json'));
  const sourceRoot = path.join(temp, 'release');
  fs.cpSync(path.join(root, 'ci/native-selectors/prow/source'), sourceRoot, {recursive: true});
  const patchFile = path.join(root, 'ci/native-selectors/prow/native-selection.patch');
  execFileSync('git', ['apply', '--check', patchFile], {cwd: sourceRoot});
  execFileSync('git', ['apply', patchFile], {cwd: sourceRoot});
  let changes = 0;
  const configured = [];
  for (const [file, digest] of Object.entries(evidence.sources)) {
    const before = read(`ci/native-selectors/prow/source/${file}`);
    assert.equal(hash(before), digest);
    const old = YAML.parse(before), updated = YAML.parse(fs.readFileSync(path.join(sourceRoot, file), 'utf8'));
    for (let i = 0; i < old.tests.length; i++) {
      const a = old.tests[i], b = updated.tests[i];
      if (a.run_if_changed !== b.run_if_changed || a.skip_if_only_changed !== b.skip_if_only_changed) {
        configured.push({file, ...b}); changes++;
        if (a.run_if_changed === undefined) delete b.run_if_changed; else b.run_if_changed = a.run_if_changed;
        if (a.skip_if_only_changed === undefined) delete b.skip_if_only_changed; else b.skip_if_only_changed = a.skip_if_only_changed;
      }
    }
    assert.deepEqual(updated, old, `${file}: only native path selectors may change`);
  }
  const pathMatch = (job, paths) => !!job.run_if_changed && paths.some(file => new RegExp(job.run_if_changed).test(file));
  const vm = configured.filter(job => job.as === 'vm-scanning-e2e-tests');
  assert.equal(vm.length, 2);
  const vmCases = [
    [['compliance/virtualmachines/roxagent/main.go'], true],
    [['sensor/common/virtualmachine/vmscraper/scraper.go', 'ui/package.json', 'README.md'], true],
    [['central/convert/internaltostorage/virtual_machine_v2.go'], true],
    [['pkg/concurrency/context.go'], true], [['scanner/indexer/repositorytocpefetcher.go'], true],
    [['operator/bundle/manifests/platform.stackrox.io_securedclusters.yaml'], true], [['scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py'], true],
    [['roxctl/main.go'], true], [['ROXIE_VERSION'], true], [['scripts/roxie.sh'], true],
    [['tests/vm_scanning_suite_test.go'], true], [['go.mod'], true], [['elsewhere/schema.proto'], true], [['central/reports/README.md'], true],
    [['ui/package.json', 'README.md'], false], [['README.md', 'OWNERS'], false], [['unrelated/vm-name.txt'], false], [[], false],
  ];
  for (const job of vm) {
    assert.equal(job.always_run, false); assert.equal(job.optional, true);
    for (const [paths, want] of vmCases) assert.equal(pathMatch(job, paths), want, paths.join(','));
  }
  const qa = configured.find(job => job.as === 'gke-qa-e2e-tests');
  const nongroovy = configured.find(job => job.as === 'gke-nongroovy-e2e-tests');
  assert(qa && nongroovy);
  assert(pathMatch(qa, ['go.mod', 'README.md']));
  assert(!pathMatch(qa, ['README.md']));
  assert(pathMatch(nongroovy, ['elsewhere/schema.proto', 'README.md']));
  assert(!pathMatch(nongroovy, ['roxctl/main.go']), 'Prow nongroovy has no positive roxctl rule in the baseline');
  const master = YAML.parse(fs.readFileSync(path.join(sourceRoot, 'ci-operator/config/stackrox/stackrox/stackrox-stackrox-master.yaml'), 'utf8'));
  assert.equal(master.tests.find(job => job.as === 'gke-ui-e2e-tests').always_run, true, 'required UI remains unconditional');
  console.log(`${changes} selector-only external edits checked, including VM mixed changes, shared dependencies, conditional QA/nongroovy and required UI preservation. No Prow controller or CI jobs ran.`);
  if (process.env.NATIVE_PROW_JOBS_GO) {
    const source = fs.readFileSync(process.env.NATIVE_PROW_JOBS_GO, 'utf8');
    assert.equal(hash(source), evidence.matcher.jobs_go_sha256, 'Prow experiment must use the pinned upstream source');
    const methods = [
      'func (cm RegexpChangeMatcher) CouldRun(', 'func (cm RegexpChangeMatcher) ShouldRun(',
      'func (cm RegexpChangeMatcher) RunsAgainstChanges(', 'func (ps Presubmit) CouldRun(', 'func (ps Presubmit) ShouldRun(',
    ].map(signature => {
      const start = source.indexOf(signature); assert(start >= 0);
      return source.slice(start, source.indexOf('\n}', start) + 2);
    }).join('\n\n');
    const experiment = path.join(temp, 'prow-matcher'); fs.mkdirSync(experiment);
    fs.writeFileSync(path.join(experiment, 'go.mod'), 'module native-prow-matcher\n\ngo 1.26.7\n');
    // Only branch eligibility is a fixed fixture; the five matcher methods are verbatim upstream code.
    const harness = String.raw`
package config
import ("errors"; "regexp"; "testing")
type ChangedFilesProvider func() ([]string, error)
type RegexpChangeMatcher struct {RunIfChanged, SkipIfOnlyChanged string; reChanges *regexp.Regexp}
type Brancher struct{}
func (Brancher) ShouldRun(string) bool {return true}
type Presubmit struct {Brancher; RegexpChangeMatcher; AlwaysRun bool}
func TestNativeMatcher(t *testing.T) {
 tests:=map[string]struct{pattern string; skip, always, force, fetchError bool; files []string; want, wantError bool}{
  "VM feature mixed with unrelated files":{pattern:VM_PATTERN,files:[]string{"sensor/common/virtualmachine/vmscraper/scraper.go","README.md"},want:true},
  "shared dependency":{pattern:VM_PATTERN,files:[]string{"pkg/concurrency/context.go"},want:true},
  "irrelevant paths":{pattern:VM_PATTERN,files:[]string{"README.md"}},
  "empty diff":{pattern:VM_PATTERN},
  "explicit test bypasses paths":{pattern:VM_PATTERN,force:true,files:[]string{"README.md"},want:true},
  "explicit test bypasses fetch failure":{pattern:VM_PATTERN,force:true,fetchError:true,want:true},
  "conditional fetch failure propagates":{pattern:VM_PATTERN,fetchError:true,wantError:true},
  "manual job stays dormant":{files:[]string{"go.mod"}},
  "manual job explicit request":{force:true,want:true},
  "test all remains conditional for docs":{pattern:BROAD_PATTERN,files:[]string{"README.md"}},
  "test all matches broad dependencies":{pattern:BROAD_PATTERN,files:[]string{"go.mod","README.md"},want:true},
  "required always-run UI ignores diff failure":{always:true,fetchError:true,want:true},
  "any UI selects UI alone":{pattern:"^ui/",files:[]string{"ui/package.json"},want:true},
  "any UI also selects a mixed diff":{pattern:"^ui/",files:[]string{"ui/package.json","OWNERS"},want:true},
  "skip-only UI excludes UI alone":{pattern:"^ui/",skip:true,files:[]string{"ui/package.json"}},
  "skip-only UI permits a mixed diff":{pattern:"^ui/",skip:true,files:[]string{"ui/package.json","OWNERS"},want:true},
 }
 for name,test:=range tests {t.Run(name,func(t *testing.T){
  matcher:=RegexpChangeMatcher{}
  if test.pattern!="" {matcher.reChanges=regexp.MustCompile(test.pattern);if test.skip {matcher.SkipIfOnlyChanged=test.pattern}else{matcher.RunIfChanged=test.pattern}}
  job:=Presubmit{RegexpChangeMatcher:matcher,AlwaysRun:test.always}
  got,err:=job.ShouldRun("master",func()([]string,error){if test.fetchError{return nil,errors.New("unavailable")};return test.files,nil},test.force,false)
  if got!=test.want||(err!=nil)!=test.wantError {t.Fatalf("run=%v error=%v; want run=%v error=%v",got,err,test.want,test.wantError)}
 })}
}
`.replaceAll('VM_PATTERN', JSON.stringify(vm[0].run_if_changed)).replaceAll('BROAD_PATTERN', JSON.stringify(qa.run_if_changed));
    const generated = path.join(experiment, 'matcher_test.go');
    fs.writeFileSync(generated, source.slice(0, source.indexOf('package config')) + harness + '\n' + methods);
    execFileSync('gofmt', ['-w', generated]);
    execFileSync('go', ['test', '-count=1', './...'], {cwd: experiment, stdio: ['ignore', 'pipe', 'pipe']});
    console.log('16 cases executed against five verbatim pinned Prow matcher methods: explicit-force recovery, conditional requests, UI all-file gap and API failures. Trigger routing and trust were not executed.');
  } else {
    console.log('Pinned upstream Prow method execution omitted; set NATIVE_PROW_JOBS_GO to the recorded jobs.go source to run that focused experiment.');
  }

} finally {
  fs.rmSync(temp, {recursive: true, force: true});
}
