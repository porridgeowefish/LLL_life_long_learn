import core from '../../../node_modules/jsxgraph/distrib/jsxgraphcore.js?raw';
import css from '../../../node_modules/jsxgraph/distrib/jsxgraph.css?raw';
import helpers from './mathTeachingRuntime.js?raw';
import license from '../../../node_modules/jsxgraph/LICENSE.MIT?raw';

// JSXGraph exports only its ESM entry. Direct distribution paths bypass that
// entry and Vite's dependency optimizer; raw source runs only in the iframe.
export const mathRuntime = {
  source: `/* JSXGraph 1.14.0 — MIT license\n${license}\n*/\n${core}\nJXG.Options.jc.compile=false;\n${helpers}`,
  css,
};
