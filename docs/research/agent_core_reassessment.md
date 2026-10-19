# Agent-only language reassessment

Date: 2026-10-19. Baseline: merged PR #24 (`4794d6b`).
Scope: agents implement, review and audit code and create deterministic tools; no human
code-review stage is assumed. Compatibility with the earlier prototype is not a requirement.

This is a design assessment informed by the implementation, not a comparative model result.
A more capable agent can identify hypotheses and implementation defects; that does not
establish that its preferred syntax will outperform alternatives. Measure the proposals
using the [research protocol](agent_language_iteration.md).

Current implementation addendum: Phase 13 implements the explicit functional subset in
[functional-core.md](../functional-core.md): typed callables, explicit partial application,
literal defaults selected by a visible marker, and eager pure collection transforms.
Known return-type mismatches now fail checking and runtime callable contracts are validated.
Tagged absence/unit, exhaustive variants, schema checks and complete effect/concurrency
semantics remain proposals or known gaps. These changes have not been benchmarked against
alternative languages or model versions.

## Main conclusion

Optimize the full agent/tool interaction around explicit structure and enforceable semantics.
The previous design put too much confidence in visual regularity, a small number of surface
forms and inspectable Go. Those can be useful, but none establishes correct interpretation,
local edits, sound audits or a faithful executable contract.

Human-oriented editor polish and source presentation are not primary optimization targets
for this experiment. Familiar source syntax can still benefit agents through prior training;
removing a human constraint does not itself justify replacing text with JSON or compressed symbols.

## Decisions and status

| Decision | Reason | Status |
| --- | --- | --- |
| Require `when (subject) { ... }` | Both the expression and case list have explicit boundaries; record literals need no context-dependent parsing exception | Implemented, breaking |
| Preserve literal contents during formatting | Canonicalization must preserve behavior; idempotence alone is insufficient | Implemented bug fix |
| Keep block functions and explicit returns for now | Clear exit sites and one existing convention; no measured reason to reintroduce expression-body aliases | Retained |
| Keep results, absence, effects and concurrency conceptually separate | Agents/tools can ask independent questions about each | Retained concepts; existing enforcement remains incomplete |
| Introduce tagged optionals and a distinct unit value | Absence, an empty result and failure should not share an untagged representation | Recommended next semantic change; syntax below is a proposal |
| Add typed variants and exhaustive result/error matching | Auditors need to inspect and recover from failures, not just abort execution or compare strings | Recommended next semantic change; not implemented |
| Add resolved structural queries/editing | Local reasoning should not require rereading and rediscovering the whole program | Proposed agent/tool interface |
| Require keyed value arguments | Roles are explicit at every call and key sets can be checked | Implemented, breaking; comparative benefit remains to be measured |
| Compare checkable contracts | Explicit invariants may improve independent validation | Experimental candidate |

## Implemented syntax change: explicit case boundaries

Old form, now rejected:

```text
when req.path
  "/health" => { return Ok(http.text(200, "ok")) }
  _ => { return Ok(http.not_found()) }
```

Current form:

```ki
when (req.path) {
  "/health" => { return Ok(value : http.text(code : 200, body : "ok")) }
  _ => { return Ok(value : http.not_found()) }
}
```

The old runtime parser read cases until the enclosing block ended. It had no delimiter
for finishing the case list and continuing with another statement. This was an actual
compositional defect, not a preference about visual style.

Parentheses around the subject deliberately add two tokens. They let the parser read any
subject expression before the case block, including a record literal or a call containing
one, without guessing whether `{` starts that expression or the case list. This is a small
grammar/locality trade-off, not a claim of lower token cost.

Tests cover continuation after a case block, nesting, selectors, record-containing calls,
parenthesized record expressions and rejection of the old form. Frontend validation here
checks delimiters; it is not a replacement for a complete body/type checker. The current
runtime's `when` still performs value dispatch, not exhaustive typed variant matching.

### Formatting is a semantic operation

The old formatter collapsed whitespace and rewrote operator-like text inside strings:
`"A  B = > C . D"` could change value. It now normalizes non-literal tokens separately and
preserves string contents. Retain canonical formatting, but test token/literal preservation
and execution behavior in addition to formatting idempotence. Full semantics-preservation
for every construct is not established by this fix.

## Recommended next semantic change: tagged absence, unit and typed errors

Current `nil` doubles as an absent optional value and a no-value function return. Current
optionals also permit bare values in examples without explicit construction, and the
implementation does not validate narrowing. That is too weak a foundation for independent
automated audits.

Proposed vocabulary (not accepted as a new supported core yet):

- `Option[T]` with `Some(value)` and `None` for absence.
- `Result[T,E]` with `Ok(value)` and `Err(error)` for fallibility.
- `unit` and `()` for successful completion without a payload.
- Typed error variants, with exhaustive matching that binds each variant's payload.

Illustrative proposed syntax only:

```text
fn lookup(id:UserId) -> Result[Option[User],LookupError] !fs {
  ...
}

fn audit(event:Event) -> Result[unit,AuditError] !log {
  ...
  return Ok(())
}

when (lookup(id)) {
  Ok(Some(user)) => { ... }
  Ok(None) => { ... }
  Err(error) => { ... }
}
```

This needs a real type core and pattern bindings, not cosmetic spelling replacements.
Renaming `R` to `Result` on its own would accomplish little; the point is an explicit,
checkable algebra of outcomes and reusable generic type rules. The names and token costs
can be evaluated after the semantics exist.

Keep postfix `?` as a candidate for early return of the same typed failure. It must be
defined as ordinary compositional propagation, not a runtime abort; its allowed enclosing
return type must be checked. Do not overload it for optional extraction or implicit recovery.

Typed identifiers and constrained values are also worth investigating: `OrderId` should
not be freely interchangeable with another string identifier, and a positive quantity
should require validation at construction. Choose checked constructors/nominal types
before inventing general theorem-proving syntax.

## What the earlier simplification did not establish

### Fewer surface forms is a useful default, not the optimization objective

One function form reduces the guide and implementation surface. It does not prove that
explicit `return` always beats a final expression, or that every convenience reduces agent
reliability. Keep the existing function convention while testing alternatives in isolation.
Do not restore multiple aliases merely to offer choice.

### Short source does not guarantee a cheap development loop

Hidden conversions, positional arguments with the same type, implicit expression sequencing
and global lookup can save tokens while adding opportunities for silent mistakes. Prefer
rejecting ambiguity over silently taking the last expression or coercing a value. Strict
expression validation is a follow-up; it is not fully implemented by this change.

Named value arguments are now required throughout the executable surface, including unary
built-ins. The quote service uses:

```text
pricing.quote(sku:sku, addon:addon, delivery:delivery, coupon:coupon)
```

The checker validates key sets for resolved calls and the runtime validates the actual callable
before evaluating its arguments. Expressions evaluate once in source order. Built-ins and
constructors use the same keyed rule; parameterless calls and positional generic type arguments
remain unchanged. The compact manifest records exact built-in keys. Full argument type checking
is still future work. Compare semantic mistakes and total repair cost with the earlier positional
form; adopting the syntax is not evidence of a measured performance advantage. Structural
queries, distinct nominal types and reusable request records remain complementary approaches.

### Explicit at interfaces does not require annotation everywhere

Keep declared parameter/return types and effect boundaries. Deterministic inference for
local expressions can reduce redundant annotations while structural tools expose inferred
types. What an audit needs to inspect need not all be repeated in authored source text.

### A single directory layout is guidance, not language semantics

The existing service layout is a useful default for this experiment. A small program should
not need artificial folders solely to satisfy an agent convention. Resolved modules and
dependencies are more fundamental than paths chosen for human navigation. This change
does not introduce a second scaffolding template or remove the current one.

### Effects and concurrency need enforceable meaning

Keep operational effects distinct from failure. An effect annotation is not a resource-level
authorization policy. A later capability design could distinguish reading one store from
unrestricted filesystem access, but it needs its own semantics and runtime enforcement.

Keep visible task creation, joining and scope. Define ownership, cancellation and early-exit
behavior before presenting `group` as an enforced lifecycle. No concurrency keyword rename
fixes those missing guarantees.

## Agent/tool interfaces

Do not make the agent reconstruct everything from formatted source. Build machine-readable
queries over the same resolved program used for checking/execution, for example:

- a symbol's typed signature, callers, callees and inferred/declarative effects;
- the affected contracts and tests for a proposed change;
- a stable diagnostic code, source/node identity, expected/actual values and evidence;
- a structural edit with a source revision/precondition, followed by revalidation.

These are proposed interfaces, not currently supported CLI commands. Use canonical text as
the current serialization. A typed IR can become the common program representation without
forcing every model to author verbose JSON ASTs. Measure text and structured editing against
the same task and include schema/tool-learning costs.

## Layered verification: deliberate differences in goals

The user's “Swiss cheese” strategy is the working verification model. Layers can catch
different mistakes without any human code reviewer. Give them different inputs, objectives
and failure probes rather than repeating the same prompt several times:

| Layer | Goal and evidence | Deliberate separation |
| --- | --- | --- |
| Specification checks | Observable requirements, invariants and boundary examples | Written before seeing candidate code |
| Compiler/static checks | Types, argument roles, effect closure and eventually exhaustiveness/lifecycles | Deterministic rules derived from the language contract |
| Adversarial agent audit | Find counterexamples, hidden assumptions and unhandled failures | Explicit falsification goal and withheld seeded defects |
| Runtime/property checks | Observe outputs, state transitions, retries and controlled fault behavior | Independently generated fixtures and properties |
| Differential/metamorphic checks | Compare implementations or related inputs with known relationships | Separate implementations/derivations and changed inputs |

Agents may build all these tools. Different goals may produce different mistakes, but do
not assume statistical independence or multiply individual detection rates. Measure which
defects each layer catches alone, overlap, false alarms, and how many escape the combined
pipeline. Use held-out defects from several categories, including evaluator and compiler
bugs, not only application bugs the implementation already handles.

Contracts such as “total equals subtotal plus shipping” could become deterministic checks
without requiring full formal verification. A contract generated from the same mistaken
requirement may still be wrong; layer diversity and measurable coverage are the experiment.

## Suggested order

1. Adopt explicit case boundaries and preserve literal semantics: this change.
2. Unify body parsing/checking/execution around a real typed representation; enforce result
   propagation, tagged values and typed decoding with negative tests.
3. Add typed variants and exhaustive matching; extend the quote service's failure paths.
4. Expose resolved queries and structured diagnostics for implementation and audit agents.
5. Measure keyed-call behavior against the earlier positional baseline and run contract and structural-editing experiments.

The unit/optional/error proposals may break the current core. There is no requirement to
preserve accidental prototype behavior, and no need to introduce all proposed syntax before
its enforcement and validation tools exist.

Phase 14 adds explicit typed stateless transducers and the canonical `list.reduce` name.
Execution and error order are explicit; see `../functional-core.md` and
`transducer_measurements.md` for memory/runtime evidence. No agent-performance advantage
is inferred from eliminating intermediate lists.
