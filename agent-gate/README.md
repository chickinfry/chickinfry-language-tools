# Agent gate

Thirty tasks that measure whether an agent can write `.chik` declarations from
the skill and the documentation alone. The user confirmed the gate on
2026-11-21: thirty tasks, at least 90% passing on the first attempt, and 100%
within two attempts. The pass verdict is taken on the Claude and GPT model
families; GLM and DeepSeek runs are reference measurements.

Nothing here runs an agent. It defines the tasks, the expected result, and a
grader that judges a submission with the same tools a customer has: the
published parser and the published checks.

## Layout

| Path | Holds |
| --- | --- |
| `tasks/<id>/task.md` | The prompt an agent receives. No answer, no hint at the file layout. |
| `tasks/<id>/expected/` | The declaration files a correct answer produces. |
| `tasks/<id>/check.json` | The grading rules, in the shape `grade.mjs` reads. |
| `grade.mjs` | Grades one submission directory and prints a result. |

## Grading

A task is graded by what the published tools say about the submission, not by
comparing text:

1. Every submitted `.chik` file must parse with no diagnostic, or with exactly
   the diagnostics `check.json` names when the task is about an error.
2. The declaration must carry the fields the task asks for. `grade.mjs` reads
   them from the parsed tree, so formatting, comments, and field order do not
   matter.
3. A task that asks for a refusal must be refused with the named code.

Tasks the published tools cannot grade yet are listed in `blocked.json` with
the product behavior that stops them. The list is re-measured with the binary
named in its note and shrinks as those behaviors are fixed.

## Why text comparison is not used

Two correct answers can differ in spacing, comments, and field order. The
corpus already pins the canonical form; a gate that also required it would
measure formatting, not whether the agent understood the language.
