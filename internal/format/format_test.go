package format

import "testing"

func TestSource(t *testing.T) {
	in := `mod main

type Resp { code:i32
body:str }

fn main()->i32= 0`
	out, err := Source(in)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}
	want := `mod main

type Resp {
  code:i32
  body:str
}

fn main() -> i32 =
  0
`
	if out != want {
		t.Fatalf("unexpected format:\n%s", out)
	}
}
