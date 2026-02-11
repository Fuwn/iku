package main

import (
	"github.com/Fuwn/iku/engine"
	"testing"
)

type ecmaScriptTestCase struct {
	name     string
	source   string
	expected string
}

var ecmaScriptTestCases = []ecmaScriptTestCase{
	{
		name: "blank lines around if block",
		source: `const x = 1;
if (x > 0) {
  doSomething();
}
const y = 2;
`,
		expected: `const x = 1;

if (x > 0) {
  doSomething();
}

const y = 2;
`,
	},
	{
		name: "blank lines around for loop",
		source: `const items = [1, 2, 3];
for (const item of items) {
  process(item);
}
const result = done();
`,
		expected: `const items = [1, 2, 3];

for (const item of items) {
  process(item);
}

const result = done();
`,
	},
	{
		name: "blank lines between different top-level types",
		source: `import { foo } from "bar";
const x = 1;
function main() {
  return x;
}
class Foo {
  bar() {}
}
`,
		expected: `import { foo } from "bar";

const x = 1;

function main() {
  return x;
}

class Foo {
  bar() {}
}
`,
	},
	{
		name: "no blank between same type",
		source: `const x = 1;
const y = 2;
const z = 3;
`,
		expected: `const x = 1;
const y = 2;
const z = 3;
`,
	},
	{
		name: "consecutive imports stay together",
		source: `import { a } from "a";
import { b } from "b";
import { c } from "c";
`,
		expected: `import { a } from "a";
import { b } from "b";
import { c } from "c";
`,
	},
	{
		name: "switch with case clauses",
		source: `const x = getValue();
switch (x) {
case "a":
  handleA();
  break;
case "b":
  handleB();
  break;
}
cleanup();
`,
		expected: `const x = getValue();

switch (x) {
case "a":
  handleA();
  break;
case "b":
  handleB();
  break;
}

cleanup();
`,
	},
	{
		name: "try catch finally",
		source: `setup();
try {
  riskyOperation();
} catch (error) {
  handleError(error);
} finally {
  cleanup();
}
done();
`,
		expected: `setup();

try {
  riskyOperation();
} catch (error) {
  handleError(error);
} finally {
  cleanup();
}

done();
`,
	},
	{
		name: "consecutive scoped blocks",
		source: `if (a) {
  doA();
}
if (b) {
  doB();
}
`,
		expected: `if (a) {
  doA();
}

if (b) {
  doB();
}
`,
	},
	{
		name: "export prefixes",
		source: `export const x = 1;
export function foo() {
  return x;
}
export class Bar {
  baz() {}
}
`,
		expected: `export const x = 1;

export function foo() {
  return x;
}

export class Bar {
  baz() {}
}
`,
	},
	{
		name: "async function",
		source: `const data = prepare();
async function fetchData() {
  return await fetch(url);
}
const result = process();
`,
		expected: `const data = prepare();

async function fetchData() {
  return await fetch(url);
}

const result = process();
`,
	},
	{
		name: "typescript interface and type",
		source: `type ID = string;
interface User {
  name: string;
  id: ID;
}
const defaultUser: User = { name: "", id: "" };
`,
		expected: `type ID = string;

interface User {
  name: string;
  id: ID;
}

const defaultUser: User = { name: "", id: "" };
`,
	},
	{
		name: "multi-line function call preserved",
		source: `const result = someFunction(
  longArgument,
  otherArgument,
);
const next = 1;
`,
		expected: `const result = someFunction(
  longArgument,
  otherArgument,
);
const next = 1;
`,
	},
	{
		name: "method chaining preserved",
		source: `const result = someArray
  .filter(x => x > 0)
  .map(x => x * 2);
const next = 1;
`,
		expected: `const result = someArray
  .filter(x => x > 0)
  .map(x => x * 2);
const next = 1;
`,
	},
	{
		name: "block comment passthrough",
		source: `/*
 * This is a block comment
 */
const x = 1;
`,
		expected: `/*
 * This is a block comment
 */
const x = 1;
`,
	},
	{
		name: "collapses extra blank lines",
		source: `const x = 1;


const y = 2;
`,
		expected: `const x = 1;
const y = 2;
`,
	},
	{
		name: "while loop",
		source: `let count = 0;
while (count < 10) {
  count++;
}
const done = true;
`,
		expected: `let count = 0;

while (count < 10) {
  count++;
}

const done = true;
`,
	},
	{
		name: "nested scopes",
		source: `function main() {
  const x = 1;
  if (x > 0) {
    for (let i = 0; i < x; i++) {
      process(i);
    }
    cleanup();
  }
  return x;
}
`,
		expected: `function main() {
  const x = 1;

  if (x > 0) {
    for (let i = 0; i < x; i++) {
      process(i);
    }

    cleanup();
  }

  return x;
}
`,
	},
	{
		name:     "template literal passthrough",
		source:   "const x = `\nhello\n\nworld\n`;\nconst y = 1;\n",
		expected: "const x = `\nhello\n\nworld\n`;\nconst y = 1;\n",
	},
	{
		name: "jsx expressions",
		source: `function Component() {
  const data = useMemo();
  if (!data) {
    return null;
  }
  return (
    <div>
      <span>{data}</span>
    </div>
  );
}
`,
		expected: `function Component() {
  const data = useMemo();

  if (!data) {
    return null;
  }

  return (
    <div>
      <span>{data}</span>
    </div>
  );
}
`,
	},
	{
		name: "expression after scoped block",
		source: `function main() {
  if (x) {
    return;
  }
  doSomething();
}
`,
		expected: `function main() {
  if (x) {
    return;
  }

  doSomething();
}
`,
	},
	{
		name: "enum declaration",
		source: `type Color = string;
enum Direction {
  Up,
  Down,
}
const x = Direction.Up;
`,
		expected: `type Color = string;

enum Direction {
  Up,
  Down,
}

const x = Direction.Up;
`,
	},
}

func TestEcmaScriptAdapter(t *testing.T) {
	for _, testCase := range ecmaScriptTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			adapter := &EcmaScriptAdapter{}
			_, events, err := adapter.Analyze([]byte(testCase.source))

			if err != nil {
				t.Fatalf("adapter error: %v", err)
			}

			formattingEngine := &engine.Engine{CommentMode: engine.CommentsFollow}
			result := formattingEngine.FormatToString(events)

			if result != testCase.expected {
				t.Errorf("mismatch\ngot:\n%s\nwant:\n%s", result, testCase.expected)
			}
		})
	}
}

func TestClassifyEcmaScriptStatement(t *testing.T) {
	cases := []struct {
		input                string
		expectedType         string
		expectedScope        bool
		expectedContinuation bool
	}{
		{"function foo() {", "function", true, false},
		{"async function foo() {", "function", true, false},
		{"export function foo() {", "function", true, false},
		{"export default function() {", "function", true, false},
		{"class Foo {", "class", true, false},
		{"export class Foo {", "class", true, false},
		{"if (x) {", "if", true, false},
		{"else if (y) {", "if", true, true},
		{"else {", "if", true, true},
		{"for (const x of items) {", "for", true, false},
		{"while (true) {", "while", true, false},
		{"do {", "do", true, false},
		{"switch (x) {", "switch", true, false},
		{"try {", "try", true, false},
		{"catch (e) {", "try", true, true},
		{"finally {", "try", true, true},
		{"interface Foo {", "interface", true, false},
		{"enum Direction {", "enum", true, false},
		{"namespace Foo {", "namespace", true, false},
		{"const x = 1;", "const", false, false},
		{"let x = 1;", "let", false, false},
		{"var x = 1;", "var", false, false},
		{"import { foo } from 'bar';", "import", false, false},
		{"type Foo = string;", "type", false, false},
		{"return x;", "return", false, false},
		{"return;", "return", false, false},
		{"throw new Error();", "throw", false, false},
		{"await fetch(url);", "await", false, false},
		{"yield value;", "yield", false, false},
		{"export const x = 1;", "const", false, false},
		{"export default class Foo {", "class", true, false},
		{"declare const x: number;", "const", false, false},
		{"declare function foo(): void;", "function", true, false},
		{"foo();", "", false, false},
		{"x = 1;", "", false, false},
		{"", "", false, false},
	}

	for _, testCase := range cases {
		statementType, isScoped, isContinuation := classifyEcmaScriptStatement(testCase.input)

		if statementType != testCase.expectedType || isScoped != testCase.expectedScope || isContinuation != testCase.expectedContinuation {
			t.Errorf("classifyEcmaScriptStatement(%q) = (%q, %v, %v), want (%q, %v, %v)",
				testCase.input, statementType, isScoped, isContinuation, testCase.expectedType, testCase.expectedScope, testCase.expectedContinuation)
		}
	}
}
