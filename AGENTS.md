# Repository guidance

- Keep changes clean and contained. Preserve existing interface and field names when the implementation can change without changing the interface.
- Breaking changes are acceptable when they simplify the design or match the requested behavior.
- Do not add migrations, compatibility paths, legacy handling, or fallback behavior for superseded formats. Update the implementation directly to the current design.
