package main

import (
	"strings"
	"testing"
)

var rustFormattingCases = []formattingCase{
	{
		name:     "statement kinds and tail returns",
		filename: "example.rs",
		source: `fn main() -> Result<(), Error> {
  let mut router = Router::new();
  let count = 1;
  router.mount("/");
  router.run()?;
  count += 1;
  if count > 1 {
    return Ok(());
  }
  Ok(())
}
`,
		expected: `fn main() -> Result<(), Error> {
  let mut router = Router::new();
  let count = 1;

  router.mount("/");
  router.run()?;

  count += 1;

  if count > 1 {
    return Ok(());
  }

  Ok(())
}
`,
	},
	{
		name:     "closure values are returns",
		filename: "example.rs",
		source: `fn main() {
  router.mount("/", |_| {
    log::info!("Hello!");
    Response::success("Hello!")
  });
  router.mount("/other", move |context| -> Response {
    let body = render(context);

    body
  });
}
`,
		expected: `fn main() {
  router.mount("/", |_| {
    log::info!("Hello!");

    Response::success("Hello!")
  });
  router.mount("/other", move |context| -> Response {
    let body = render(context);

    body
  });
}
`,
	},
	{
		name:     "top-level items",
		filename: "example.rs",
		source: `//! Crate documentation.
#![deny(missing_docs)]
#[macro_use]
extern crate log;
use std::fmt;
pub use crate::router::Router;
mod router;
pub mod response;
const LIMIT: usize = 1024;
static NAME: &str = "iku";
type Result<T> = std::result::Result<T, Error>;
pub struct Marker;
struct Point {
  x: i32,
}
fn main() {}
`,
		expected: `//! Crate documentation.
#![deny(missing_docs)]

#[macro_use]
extern crate log;

use std::fmt;
pub use crate::router::Router;

mod router;
pub mod response;

const LIMIT: usize = 1024;
static NAME: &str = "iku";

type Result<T> = std::result::Result<T, Error>;

pub struct Marker;

struct Point {
  x: i32,
}

fn main() {}
`,
	},
	{
		name:     "single-line definitions cluster unless attributed",
		filename: "example.rs",
		source: `impl Post {
  pub const fn title(&self) -> &str { &self.title }

  pub const fn link(&self) -> &str { &self.link }
  #[must_use]
  pub const fn size(&self) -> usize { self.size }
  #[must_use]
  pub const fn status(&self) -> u8 { self.status }
}
`,
		expected: `impl Post {
  pub const fn title(&self) -> &str { &self.title }
  pub const fn link(&self) -> &str { &self.link }

  #[must_use]
  pub const fn size(&self) -> usize { self.size }

  #[must_use]
  pub const fn status(&self) -> u8 { self.status }
}
`,
	},
	{
		name:     "trait signatures pack and default methods separate",
		filename: "example.rs",
		source: `trait Named {
  fn name(&self) -> String;
  fn title(&self) -> String;
  fn greeting(&self) -> String {
    format!("Hello, {}", self.name())
  }
}
`,
		expected: `trait Named {
  fn name(&self) -> String;
  fn title(&self) -> String;

  fn greeting(&self) -> String {
    format!("Hello, {}", self.name())
  }
}
`,
	},
	{
		name:     "field lists keep documentation packed",
		filename: "example.rs",
		source: `/// Options.
#[derive(Debug)]
pub enum Option {
  /// The first option.
  First,

  /// The second option.
  Second { value: u8 },
  /// The third option.
  Third {
    /// The inner value.
    inner: u8,
    /// The outer value.
    outer: u8,
  },
}
struct Limits {
  /// The connection cap.
  #[serde(default)]
  pub max_connections: usize,
  /// The timeout.
  pub timeout: Duration,
}
`,
		expected: `/// Options.
#[derive(Debug)]
pub enum Option {
  /// The first option.
  First,
  /// The second option.
  Second { value: u8 },
  /// The third option.
  Third {
    /// The inner value.
    inner: u8,
    /// The outer value.
    outer: u8,
  },
}

struct Limits {
  /// The connection cap.
  #[serde(default)]
  pub max_connections: usize,
  /// The timeout.
  pub timeout: Duration,
}
`,
	},
	{
		name:     "multiline block arms are scoped",
		filename: "example.rs",
		source: `fn render(node: &Node) -> usize {
  let size = match node {
    Node::Text(text) => text.len(),

    Node::Link { to, text } if text.is_some() => {
      let label = text.as_deref().unwrap_or(to);
      push(label);
      label.len()
    }

    Node::List(items) =>
      items.len(),
    Node::Quote(_)
    | Node::Heading { .. } => 0,
    _ => 1,
  };
  size
}
`,
		expected: `fn render(node: &Node) -> usize {
  let size = match node {
    Node::Text(text) => text.len(),

    Node::Link { to, text } if text.is_some() => {
      let label = text.as_deref().unwrap_or(to);

      push(label);

      label.len()
    }

    Node::List(items) =>
      items.len(),
    Node::Quote(_)
    | Node::Heading { .. } => 0,
    _ => 1,
  };

  size
}
`,
	},
	{
		name:     "patterns and struct literals are not bodies",
		filename: "example.rs",
		source: `fn patterns(shape: Shape, point: Point) {
  let Point { x, y } = point;
  if let Shape::Circle { radius } = shape {
    draw(radius);
  }
  for Point { x, .. } in points {
    draw(x);
  }
  let make = move |value| Point { x: value, y: value };
  let point = Point {
    x: compute(),

    y: 2,
  };
  let Some(value) = maybe else {
    return;
  };
  draw(value);
}
`,
		expected: `fn patterns(shape: Shape, point: Point) {
  let Point { x, y } = point;

  if let Shape::Circle { radius } = shape {
    draw(radius);
  }

  for Point { x, .. } in points {
    draw(x);
  }

  let make = move |value| Point { x: value, y: value };
  let point = Point {
    x: compute(),
    y: 2,
  };
  let Some(value) = maybe else {
    return;
  };

  draw(value);
}
`,
	},
	{
		name:     "blocks inside arguments and use trees",
		filename: "example.rs",
		source: `use germ::{
  ast::Node,

  convert::html,
};
fn feed(xml: &mut Writer) {
  xml.add_item(&{
    let mut builder = Item::builder();
    builder.add_field("title", "post");
    builder
  });
  let schema = Mutex::new({
    let mut builder = Schema::builder();
    builder.add_text_field("path");
    builder.build()
  });
}
`,
		expected: `use germ::{
  ast::Node,
  convert::html,
};

fn feed(xml: &mut Writer) {
  xml.add_item(&{
    let mut builder = Item::builder();

    builder.add_field("title", "post");

    builder
  });

  let schema = Mutex::new({
    let mut builder = Schema::builder();

    builder.add_text_field("path");

    builder.build()
  });
}
`,
	},
	{
		name:     "wrapped receivers take the role of their method chain",
		filename: "example.rs",
		source: `fn build() {
  certificate.set_pubkey(&key).unwrap();
  certificate
    .set_not_before(&start)
    .unwrap();
  cloned
    .options = options;
  certificate
    .field
    .sign(&key);
}
`,
		expected: `fn build() {
  certificate.set_pubkey(&key).unwrap();
  certificate
    .set_not_before(&start)
    .unwrap();

  cloned
    .options = options;

  certificate
    .field
    .sign(&key);
}
`,
	},
	{
		name:     "wrapped signatures and where clauses",
		filename: "example.rs",
		source: `pub fn fields<T, U>(
  arguments: TokenStream,
  item: syn::ItemStruct,
) -> Result<Vec<T>, Box<dyn Error>>
where
  T: Clone,
  U: Default,
{
  let first = arguments.clone();
  run(first);
}
impl<T> From<T> for Point
where
  T: Into<i32>,
{
  fn from(value: T) -> Self { Self { x: value.into() } }
}
`,
		expected: `pub fn fields<T, U>(
  arguments: TokenStream,
  item: syn::ItemStruct,
) -> Result<Vec<T>, Box<dyn Error>>
where
  T: Clone,
  U: Default,
{
  let first = arguments.clone();

  run(first);
}

impl<T> From<T> for Point
where
  T: Into<i32>,
{
  fn from(value: T) -> Self { Self { x: value.into() } }
}
`,
	},
	{
		name:     "literals and comments are verbatim",
		filename: "example.rs",
		source: `const RAW: &str = r#"first

"quoted" { fn
"#;
const TEXT: &str = "first

second {";
const CONTINUED: &str = "first \
  second";
static OPEN: char = '{';
/* outer
  /* nested */

  still outer { */
fn lifetimes<'a>(value: &'a str) -> &'a str {
  let quote = '\'';
  let brace = b'}';
  'outer: loop {
    break 'outer;
  }
  value
}
`,
		expected: `const RAW: &str = r#"first

"quoted" { fn
"#;
const TEXT: &str = "first

second {";
const CONTINUED: &str = "first \
  second";
static OPEN: char = '{';

/* outer
  /* nested */

  still outer { */
fn lifetimes<'a>(value: &'a str) -> &'a str {
  let quote = '\'';
  let brace = b'}';

  'outer: loop {
    break 'outer;
  }

  value
}
`,
	},
	{
		name:     "macro bodies are verbatim",
		filename: "example.rs",
		source: `macro_rules! square {
  ($value:expr) => {

    $value * $value
  };
}
fn expand() -> TokenStream {
  let name = format_ident!("router");
  let tokens = quote! {
    let value = 1;

    #name: value,
  };
  tokens.into()
}
`,
		expected: `macro_rules! square {
  ($value:expr) => {

    $value * $value
  };
}

fn expand() -> TokenStream {
  let name = format_ident!("router");
  let tokens = quote! {
    let value = 1;

    #name: value,
  };

  tokens.into()
}
`,
	},
	{
		name:     "blocks, labels, and nested modules",
		filename: "example.rs",
		source: `fn main() {
  let value = unsafe { abs(-1) };
  let task = async move {
    run().await;
  };
  let labelled = 'block: {
    if value > 1 {
      break 'block 1;
    }
    2
  };
  unsafe {
    release();
  }
  {
    let scoped = 1;
  }
}
#[cfg(test)]
mod tests {
  use super::*;
  #[test]
  fn works() {
    assert_eq!(value(), 2);
  }
}
`,
		expected: `fn main() {
  let value = unsafe { abs(-1) };
  let task = async move {
    run().await;
  };
  let labelled = 'block: {
    if value > 1 {
      break 'block 1;
    }

    2
  };

  unsafe {
    release();
  }

  {
    let scoped = 1;
  }
}

#[cfg(test)]
mod tests {
  use super::*;

  #[test]
  fn works() {
    assert_eq!(value(), 2);
  }
}
`,
	},
	{
		name:     "comments attach to the following statement",
		filename: "example.rs",
		source: `fn main() {
  router.set_certificate_file("public.pem");
  // The private key is optional.
  router.set_private_key_file("private.pem");
}
`,
		expected: `fn main() {
  router.set_certificate_file("public.pem");

  // The private key is optional.
  router.set_private_key_file("private.pem");
}
`,
	},
}

func TestFormatRust(t *testing.T) {
	for _, testCase := range rustFormattingCases {
		t.Run(testCase.name, func(t *testing.T) {
			assertFormatting(t, &Formatter{}, testCase)
		})
	}
}

func TestClassifyRustStatement(t *testing.T) {
	cases := []struct {
		input             string
		expectedType      string
		expectedScope     bool
		expectedAwaitItem bool
	}{
		{"fn main() {", "function", false, false},
		{"pub(crate) async unsafe fn run() {", "function", false, false},
		{"pub const fn size(&self) -> usize { self.size }", "function", false, false},
		{"extern 0   fn callback() {", "function", false, false},
		{"#[inline] fn fast() {}", "function", false, false},
		{"struct Point {", "type", true, false},
		{"pub struct Marker;", "type", true, false},
		{"enum Shape {", "type", true, false},
		{"union Bits {", "type", true, false},
		{"union.extend(other);", "call", false, false},
		{"unsafe impl Send for Router {}", "type", true, false},
		{"trait Named {", "type", true, false},
		{"type Result<T> = std::result::Result<T, Error>;", "type", false, false},
		{"mod tests {", "module", false, false},
		{"extern 0   {", "module", false, false},
		{"extern crate log;", "crate", false, false},
		{"use std::fmt;", "use", false, false},
		{"let value = 1;", "binding", false, false},
		{"const LIMIT: usize = 1;", "binding", false, false},
		{"static NAME: &str = \"iku\";", "binding", false, false},
		{"if ready {", "if", true, false},
		{"while let Some(item) = items.next() {", "while", true, false},
		{"for item in items {", "for", true, false},
		{"'outer: loop {", "loop", true, false},
		{"match value {", "match", true, false},
		{"unsafe {", "block", true, false},
		{"{", "block", true, false},
		{"return Ok(());", "return", false, false},
		{"break 'outer;", "branch", false, false},
		{"continue;", "branch", false, false},
		{"macro_rules! square {", "macro", false, false},
		{"#![allow(dead_code)]", "attribute", false, false},
		{"#[derive(Debug)]", "", false, true},
		{"#[allow(", "", false, true},
		{"count += 1;", "assignment", false, false},
		{"*value = 2;", "assignment", false, false},
		{"value == other", "expression", false, false},
		{"start..=end", "expression", false, false},
		{"router.run().await?;", "call", false, false},
		{"println!(\"{value}\");", "call", false, false},
		{"vec![1, 2]", "call", false, false},
		{"value", "expression", false, false},
	}

	for _, testCase := range cases {
		statementType, isScoped, awaitsItem := classifyRustStatement(testCase.input)

		if statementType != testCase.expectedType || isScoped != testCase.expectedScope || awaitsItem != testCase.expectedAwaitItem {
			t.Errorf("classifyRustStatement(%q) = (%q, %v, %v), want (%q, %v, %v)",
				testCase.input, statementType, isScoped, awaitsItem, testCase.expectedType, testCase.expectedScope, testCase.expectedAwaitItem)
		}
	}
}

func TestMaskRustLiteralsAndComments(t *testing.T) {
	var lexicalState rustLexicalState
	cases := []struct {
		line     string
		expected string
	}{
		{`let brace = '{'; // }`, `let brace = 0 ;`},
		{`fn borrow<'a>(value: &'a str) {`, `fn borrow<'a>(value: &'a str) {`},
		{`let escaped = '\''; let raw = r#type;`, `let escaped = 0 ; let raw = r#type;`},
		{`let text = r##"first "# {`, `let text = 0`},
		{`still raw }"## + 1;`, `+ 1;`},
		{`let text = "first`, `let text = 0`},
		{`second \" { "; done();`, `; done();`},
		{`/* outer /* inner */ still { */ code`, `code`},
		{`/* open /* nested */`, ``},
		{`still open { */ code`, `code`},
		{`let bytes = br"{" + b"}" + c"x";`, `let bytes = b0 + b0 + c0 ;`},
	}

	for _, testCase := range cases {
		if masked := strings.Join(strings.Fields(lexicalState.maskLiteralsAndComments(testCase.line)), " "); masked != testCase.expected {
			t.Errorf("maskLiteralsAndComments(%q) = %q, want %q", testCase.line, masked, testCase.expected)
		}
	}
}
