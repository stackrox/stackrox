import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { Lexer, Parser, Evaluator, data } from '@actions/expressions';

// GitHub supplies status functions and an implicit success() guard. The library
// evaluates the expression language; these fixtures supply runner status only.
for (const c of JSON.parse(readFileSync(0, 'utf8'))) {
    const functions = new Map(Object.entries({
        always: true,
        success: c.status === 'success',
        failure: c.status === 'failure',
        cancelled: c.status === 'cancelled',
    }).map(([name, value]) => [name, {
        name, minArgs: 0, maxArgs: 0,
        call: () => new data.BooleanData(value),
    }]));
    let source = c.expression.trim();
    if (source.startsWith('${{')) source = source.slice(3, -2).trim();
    const tokens = new Lexer(source || 'true').lex().tokens;
    if (!tokens.some((t, i) => functions.has(t.lexeme?.toLowerCase()) &&
        tokens[i + 1]?.lexeme === '(')) {
        source = `success() && (${source || 'true'})`;
    }
    const context = JSON.parse(JSON.stringify(c.context), data.reviver);
    const expression = new Parser(new Lexer(source).lex().tokens,
        Object.keys(c.context), [...functions.values()]).parse();
    const result = new Evaluator(expression, context, functions).evaluate();
    assert.equal(result.coerceString(), String(c.want), c.name);
}
