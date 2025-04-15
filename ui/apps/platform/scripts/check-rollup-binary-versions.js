const fs = require('fs');
const path = require('path');

const projectPkgPath = path.resolve('package.json');
const viteRollupPkgPath = path.resolve('node_modules/vite/node_modules/rollup/package.json');

if (!fs.existsSync(viteRollupPkgPath)) {
  console.warn('[check-rollup-binaries] Could not find vite’s internal rollup dependency. Skipping check.');
  process.exit(0);
}

const projectPkg = JSON.parse(fs.readFileSync(projectPkgPath, 'utf8'));
const rollupPkg = JSON.parse(fs.readFileSync(viteRollupPkgPath, 'utf8'));

const expectedBinaries = rollupPkg.optionalDependencies || {};
const projectRollupDeps = Object.entries(projectPkg.optionalDependencies || {}).filter(([name]) =>
  name.startsWith('@rollup/rollup-')
);

const mismatches = [];

for (const [pkg, declaredVersion] of projectRollupDeps) {
  const expectedVersion = expectedBinaries[pkg];
  if (!expectedVersion) {
    mismatches.push({
      pkg,
      declared: declaredVersion,
      expected: '(missing from vite’s rollup)',
    });
  } else {
    const normalizedDeclared = declaredVersion.replace(/^[^\d]*/, '');
    const normalizedExpected = expectedVersion.replace(/^[^\d]*/, '');

    if (normalizedDeclared !== normalizedExpected) {
      mismatches.push({
        pkg,
        declared: declaredVersion,
        expected: expectedVersion,
      });
    }
  }
}

if (mismatches.length > 0) {
  console.warn('\n⚠️ [check-rollup-binaries] Detected version mismatches for Rollup platform binaries:');
  mismatches.forEach(({ pkg, declared, expected }) => {
    console.warn(`  - ${pkg} is pinned to ${declared}, but Vite's Rollup wants ${expected}`);
  });
  console.warn('\n👉 To fix: Update the optionalDependencies in `package.json` and run `npm install`');
} else {
  console.log('✅ [check-rollup-binaries] All Rollup platform binaries match Vite’s internal Rollup.');
}
