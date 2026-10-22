import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import {execFileSync} from 'node:child_process';
import {Lexer, Parser, Evaluator, data} from '@actions/expressions';
import YAML from 'yaml';

const root = path.resolve('../..');
const read = file => fs.readFileSync(path.join(root, file), 'utf8');
const workflow = file => YAML.parse(read(`.github/workflows/${file}`));
const selection = workflow('native-ci-selection.yaml').jobs.select;
const dispatcher = workflow('e2e-dispatch.yaml').jobs['should-dispatch'];
const scenarios = JSON.parse(fs.readFileSync('testdata/scenarios.json'));
const expected = JSON.parse(fs.readFileSync('testdata/expected.json'));
const inventory = JSON.parse(fs.readFileSync('baseline.json')).inventory;
const statusFunctions = ['always', 'cancelled', 'success', 'failure'].map(name => ({
  name, minArgs: 0, maxArgs: 0,
  call: () => new data.BooleanData(name === 'always' || name === 'success'),
}));
const astCache = new Map();
function evaluate(source, context) {
  let expression = source.trim().replace(/^\$\{\{\s*|\s*\}\}$/g, '');
  let parsed = astCache.get(expression);
  if (!parsed) {
    parsed = new Parser(new Lexer(expression).lex().tokens,
      ['github', 'inputs', 'needs', 'steps', 'matrix', 'env', 'runner', 'secrets', 'vars'], statusFunctions).parse();
    astCache.set(expression, parsed);
  }
  return new Evaluator(parsed, JSON.parse(JSON.stringify(context), data.reviver),
    new Map(statusFunctions.map(f => [f.name, f]))).evaluate().coerceString();
}
function evaluateEnv(step, context) {
  return Object.fromEntries(Object.entries(step.env).map(([name, expr]) => [name, evaluate(expr, context)]));
}
const factsBatch = JSON.parse(execFileSync('python3', ['-c', `
import json, sys
from path_facts import classify
patterns=json.load(open('paths.json'))
print(json.dumps([classify(case['files'], patterns) for case in json.load(sys.stdin)]))
`], {input: JSON.stringify(scenarios), encoding: 'utf8'}));
const results = new Map();
for (const [index, scenario] of scenarios.entries()) {
  const labels = [...scenario.labels, ...(scenario.enforced ? ['ci-dispatcher-enforce'] : [])];
  const context = {
    github: {event_name: scenario.event, event: {pull_request: {
      labels: labels.map(name => ({name})), draft: scenario.draft,
      head: {repo: {fork: scenario.fork}},
    }}},
    inputs: {'live-labels': true},
    steps: {
      paths: {outputs: scenario.unknown || scenario.event !== 'pull_request' ? {} : {
        any: JSON.stringify(factsBatch[index].any), every: JSON.stringify(factsBatch[index].every), known: 'true',
      }},
      live: {outputs: {labels: JSON.stringify(labels), known: 'true'}},
    },
  };
  const common = evaluateEnv(selection.steps.find(s => s.id === 'context'), context);
  context.steps.context = {outputs: Object.fromEntries(Object.entries(common).map(([k, v]) => [k.toLowerCase(), v]))};
  const choices = evaluateEnv(selection.steps.find(s => s.id === 'policy'), context);
  context.steps.policy = {outputs: Object.fromEntries(Object.entries(choices).map(([k, v]) => [k.slice(7).replaceAll('_', '-'), v]))};
  const outputs = Object.fromEntries(Object.entries(selection.outputs).map(([k, v]) => [k, evaluate(v, context)]));
  const selected = inventory.filter(job => outputs[job] === 'true').sort();
  if (scenario.event === 'pull_request') {
    assert.deepEqual(selected, expected[scenario.name].selected, `resolver parity: ${scenario.name}`);
  }
  const uiOnly = scenario.files.length > 0 && scenario.files.every(p => p.startsWith('ui/'));
  context.needs = {
    'native-selection': {outputs, result: 'success'},
    'detect-changes': {outputs: {ui_only: uiOnly ? 'true' : '', files: scenario.files.map(p => `^${p}$`).join('\n')}, result: 'success'},
  };
  context.env = evaluateEnv(dispatcher, context);
  const dispatch = evaluateEnv(dispatcher.steps.find(s => s.id === 'dispatch'), context);
  const jobs = Object.fromEntries(Object.entries(dispatch).map(([k, v]) => [k.slice(7).replaceAll('_', '-'), v === 'true']));
  context.steps.dispatch = {outputs: {jobs: JSON.stringify(jobs)}};
  const waits = evaluateEnv(dispatcher.steps.find(s => s.id === 'waits'), context);
  assert.deepEqual(jobs, JSON.parse(expected[scenario.name].dispatch.jobs), `dispatch parity: ${scenario.name}`);
  assert.equal(waits.WAIT_GHA, expected[scenario.name].dispatch['wait-for-gha-images'], `GHA wait: ${scenario.name}`);
  assert.equal(waits.WAIT_KONFLUX, expected[scenario.name].dispatch['wait-for-konflux-images'], `Konflux wait: ${scenario.name}`);
  results.set(scenario.name, context);
}

// Ordinary gates use event labels. The dispatcher refreshes labels and disables
// enforcement if that refresh fails, even when the event requested enforcement.
const stale = structuredClone(results.get('docs'));
stale.github.event.pull_request.labels = [{name: 'ci-dispatcher-enforce'}];
stale.steps.live = {outputs: {}};
assert.equal(evaluateEnv(selection.steps.find(s => s.id === 'context'), stale).ENFORCED, 'false');
stale.inputs['live-labels'] = false;
assert.equal(evaluateEnv(selection.steps.find(s => s.id === 'context'), stale).ENFORCED, 'true');
stale.inputs['live-labels'] = true;
stale.steps.live = {outputs: {known: 'true', labels: '[]'}};
assert.equal(evaluateEnv(selection.steps.find(s => s.id === 'context'), stale).ENFORCED, 'false');
stale.steps.live.outputs.labels = '["ci-dispatcher-enforce","ci-run-all-tests"]';
assert.equal(evaluateEnv(selection.steps.find(s => s.id === 'context'), stale).ALL, 'true');

// Evaluate the actual consumer gates, including cleanup steps and the matrix jobs.
let gatedSteps = 0;
for (const file of ['build.yaml', 'unit-tests.yaml', 'style.yaml', 'check-crd-compatibility.yaml', 'scanner-db-integration-tests.yaml']) {
  const doc = workflow(file);
  assert(doc.on.pull_request.types.includes('labeled'), file);
  assert(doc.on.pull_request.types.includes('unlabeled'), file);
  assert(!doc.on.pull_request.paths && !doc.on.pull_request['paths-ignore'], file);
  for (const [name, job] of Object.entries(doc.jobs)) {
    if (!JSON.stringify(job).includes('needs.native-selection.outputs')) continue;
    const context = structuredClone(results.get('docs'));
    for (const dependency of [].concat(job.needs || [])) {
      context.needs[dependency] ??= {result: 'success', outputs: {}};
    }
    context.vars = {CI_NATIVE_ALLOW_IMAGE_SKIPS: 'true'};
    context.matrix = {}; context.inputs = {}; context.env = {}; context.steps = {};
    if (job.strategy) {
      assert(!job.if.includes('outputs.enforced'), `matrix names must survive: ${name}`);
    }
    if (file === 'build.yaml' || job.strategy) {
      for (const step of job.steps) {
        assert(step.if?.includes('needs.native-selection.outputs'), `${file}/${name}: ungated step`);
        assert.equal(evaluate(step.if, context), 'false', `${file}/${name}: skipped work`);
        if (file === 'build.yaml') {
          context.vars = {};
          // An unset readiness variable removes only the image-selection gate.
          const original = step.if.replace(/\(vars\.CI_NATIVE_ALLOW_IMAGE_SKIPS[^)]*\)/, 'true');
          assert.equal(evaluate(step.if, context), evaluate(original, context), `${name}: images retained`);
          context.vars = {CI_NATIVE_ALLOW_IMAGE_SKIPS: 'true'};
        }
        gatedSteps++;
      }
    }
    // Missing selector output must preserve the original outer conditions.
    context.needs['native-selection'] = {result: 'failure', outputs: {}};
    if (name === 'go' || name === 'go-postgres' || (file === 'build.yaml' && name !== 'scan-go-binaries')) {
      assert.equal(evaluate(job.if, context), 'true', `${file}/${name}: infrastructure fallback`);
    }
  }
}
const e2e = workflow('e2e-dispatch.yaml');
for (const name of ['wait-for-images', 'wait-for-konflux-images']) {
  assert(!e2e.jobs[name].if, `wait check must stay present: ${name}`);
  assert(e2e.jobs[name].steps.every(s => s.if?.includes('needs.should-dispatch.outputs')));
}
for (const [name, job] of Object.entries(e2e.jobs).filter(([name]) => name.startsWith('e2e-'))) {
  assert(!job.if, `required child checks need the reusable call: ${name}`);
  assert(job.with['should-run'].includes('needs.should-dispatch.outputs.jobs'));
}
const required = JSON.parse(fs.readFileSync('testdata/required-gha-checks.json'));
let contexts = 0;
for (const [file, jobs] of Object.entries(required)) {
  const doc = workflow(file);
  for (const [name, checks] of Object.entries(jobs)) {
    assert(doc.jobs[name], `${file}/${name}: missing required check job`);
    if (doc.jobs[name].strategy) assert(!doc.jobs[name].if?.includes('outputs.enforced'));
    contexts += checks.length;
  }
}
assert.equal(contexts, 22);

// Execute the existing matrix builder with fixture context functions. It only
// constructs JSON; no build, registry operation, or GitHub request runs here.
const build = workflow('build.yaml');
for (const name of ['pre-build-ui', 'pre-build-docs', 'pre-build-oss-notice', 'build-operator-bundle']) {
  assert.deepEqual(build.jobs[name].needs, ['native-selection'], `${name}: preserve independent prerequisites`);
}
const matrixScript = build.jobs['define-job-matrix'].steps.find(s => s.id === 'define-job-matrix').run;
const matrixCases = [
  {name: 'PR', pr: true, tagged: false, labels: []},
  {name: 'expanded PR', pr: true, tagged: false, labels: ['ci-build-all-arch', 'ci-build-prerelease', 'ci-build-race-condition-debug']},
  {name: 'merge', pr: false, tagged: false, labels: []},
  {name: 'nightly tag', pr: false, tagged: true, labels: []},
];
const expandMatrix = matrix => {
  let rows = [{}];
  for (const [key, values] of Object.entries(matrix).filter(([key]) => key !== 'exclude' && key !== 'include')) {
    rows = rows.flatMap(row => values.map(value => ({...row, [key]: value})));
  }
  return rows.filter(row => !matrix.exclude?.some(excluded => Object.entries(excluded).every(([key, value]) => row[key] === value)));
};
const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'native-matrix-'));
try {
  for (const scenario of matrixCases) {
    const output = path.join(temp, 'output');
    fs.writeFileSync(output, '');
    const fixtures = `is_in_PR_context() { ${scenario.pr}; }\nis_tagged() { ${scenario.tagged}; }\npr_has_label() { case "$1" in ${scenario.labels.length ? scenario.labels.join('|') : '__none__'}) return 0;; *) return 1;; esac; }`;
    const script = matrixScript.replace("source './scripts/ci/lib.sh'", fixtures);
    assert.notEqual(script, matrixScript, 'matrix fixture must replace the external context reader');
    execFileSync('bash', ['-euo', 'pipefail', '-c', script], {env: {...process.env, GITHUB_OUTPUT: output}});
    const matrices = JSON.parse(fs.readFileSync(output, 'utf8').trim().slice('matrix='.length));
    for (const [name, checks] of Object.entries(required['build.yaml'])) {
      const reference = build.jobs[name].strategy.matrix.match(/\.([a-z_]+)\s*}}$/)[1];
      const names = expandMatrix(matrices[reference]).map(row => `${name} (${Object.values(row).join(', ')})`);
      for (const check of checks) assert(names.includes(check), `${scenario.name}: required variant ${check}`);
    }
  }
} finally {
  fs.rmSync(temp, {recursive: true, force: true});
}
console.log(`${scenarios.length} pinned baseline scenarios passed; ${gatedSteps} gated steps and ${contexts} required GHA contexts checked.`);
console.log('Four build matrices preserve required variants; label refresh and image-readiness fallbacks passed.');
