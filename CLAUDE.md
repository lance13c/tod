# Test God - Tod - Agentic Manual Tester

## Year
We are in the year 2025.
Never say best practices 2024, always say best practices 2025 if you this comes up.

## Truth Seeking
You actually check what I say and find the truth! No sycophantic behavior. You must assume I might be wrong. I might have a good point. You need to think about it.

## Overview

An agentic TUI that act as a developer's personal agent of beautiful software where UX and art come first. We prioritize emotion, attention to detail, and user experience over pure functionality. This monorepo contains multiple applications that share a unified backend and database.

## Claude
- When you are conducting reviews, there might be nothing wrong and that is okay, there also might be improvements you see. The user (me), is not always right and you might be wrong. Take everything with a grain of salt, be curious, looks up docs, and ask questions. We both might be wrong.
- Avoid writing technical implementation into the specs, and instead keep it in english if possible.

## Development Server
- NEVER run dev server, always assume it is running

## Development Workflow
- After ui edits to the desktop app check the tarui app console logs to see if there are any errors, if so fix them please

## URL and Environment Configuration
- **NEVER use fallback URLs** - always require proper environment variables
- **Bad practice**: `const url = process.env.API_URL || 'http://localhost:3000'`
- **Good practice**: `const url = process.env.API_URL` with proper error handling if missing
- Environment variables should be explicitly set and validated, not assumed with defaults

## Deployment Rules
- NEVER deploy to production unless explicitly instructed
- Always ask before deploying to production
- Staging deployments can be done without asking

## Philosophy

**Core Values:**
- Beautiful software where UX and art come first
- Attention to detail is paramount

## Icon Management
- We often have issues with icons, ensure we add icons to the icon map and relevant files when we add or update them

## Circular Dependency Detection

### Tool: dpdm (Detect Project Dependencies Map)
We use `dpdm` to detect circular dependencies in our TypeScript/JavaScript codebase. This tool is preferred over alternatives like `madge` because:
- Uses TypeScript AST for accurate parsing
- Better handles mixed JS/TS codebases  
- Provides clear circular dependency chain output
- Fast performance even on large projects

### Usage
After making code changes and running lint, check for circular dependencies:
```bash
# Check entire monorepo for circular dependencies
pnpm circular:check

# Run lint and circular dependency check together
pnpm lint:all
```

### Understanding the Output
When circular dependencies are found, dpdm will show:
- The files involved in the circular dependency chain
- The import path that creates the cycle
- Exit with code 1 to fail CI/CD pipelines

Example output:
```
✖ Circular dependency found: 
  src/services/auth.ts → src/contexts/AuthContext.tsx → src/services/auth.ts
```

### Fixing Circular Dependencies
Common strategies:
1. **Extract shared types/interfaces** to a separate file
2. **Use dependency injection** instead of direct imports
3. **Lazy load modules** with dynamic imports
4. **Refactor to remove bidirectional dependencies**

### Important
- Always run `pnpm circular:check` after `pnpm lint`
- The check covers both `/apps` and `/packages` directories
- Circular dependencies will cause build failures and runtime errors

### NextJS
- Avoid use nextjs API, always use the apps/api

## Blog Writing Guidelines

### Numbered List Formatting
When writing blog content with numbered lists, avoid using standard markdown numbered lists (`1. **Item**`) as they can have rendering issues. Instead, use this format:

**✅ Correct format:**
```markdown
**1. Item title**  
Description text goes here.

**2. Another item**  
More description text.
```

**❌ Avoid:**
```markdown
1. **Item title**
   Description text goes here.

2. **Another item**
   More description text.
```

**Why:** Standard markdown numbered lists can cause rendering issues where numbers appear on separate lines from headings, or text runs together on one line. The bold paragraph format ensures consistent display with numbers inline with headings and proper line breaks.

### General Writing Style
- Use "collabs" for informal references to collaborations (industry standard)
- Avoid excessive hyphenation that sounds AI-generated
- Keep bullet points concise with consistent formatting
- Use tables with proper alignment (left for text, right for numbers)
