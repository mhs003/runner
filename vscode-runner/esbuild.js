const esbuild = require('esbuild');

const watch = process.argv.includes('--watch');

/** @type {esbuild.BuildOptions} */
const config = {
  entryPoints: ['src/extension.ts'],
  bundle: true,
  outfile: 'out/extension.js',
  external: ['vscode'],
  platform: 'node',
  format: 'cjs',
  sourcemap: true,
  minify: false,
};

if (watch) {
  esbuild.context(config).then(ctx => ctx.watch());
} else {
  esbuild.buildSync(config);
}
