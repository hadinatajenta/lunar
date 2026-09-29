# Global Rule: No Code Comments in Generated Code

When generating or modifying code for both Frontend and Backend across all files and components:

1. **DO NOT USE COMMENTS**: Never write inline comments (`// ...`, `/* ... */`, `# ...`, `<!-- ... -->`) in generated or updated code.
2. **Self-Explanatory Code**: Code must be clean, readable, and self-documenting through expressive naming, explicit types, and cohesive functions.
3. **No Redundant Explanations**: Do not explain what a line of code does in comments.
4. **Preserve Unrelated Comments**: Do not aggressively delete existing legacy comments in untouched files, but never emit new comments in your generated code.
