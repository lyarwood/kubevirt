# KubeVirt release-1.8 AI Attribution Review

Review of AI attribution compliance on the `release-1.8` branch against the
[KubeVirt AI Contribution Policy](https://github.com/kubevirt/community/blob/main/ai-contribution-policy.md).

Commit range: `release-1.7..release-1.8` (the v1.8.0 development cycle).

## Overview

| Metric | Count |
|--------|-------|
| Total commits | 1,089 |
| Merge commits | 351 |
| Non-merge commits | 738 |
| Commits with AI attribution | 76 |
| AI-attributed share (of non-merge) | 10.3% |
| Distinct contributors using AI | 15 |
| DCO (Signed-off-by) compliance | 76/76 (100%) |

## AI Tools Used

| Tool | Trailer Instances | Authors |
|------|-------------------|---------|
| Claude (all variants) | 70 | 11 (8 Claude-only, 3 also use Cursor) |
| Cursor | 14 | 7 (4 Cursor-only, 3 also use Claude) |

Claude dominates at ~83% of all AI attribution instances.

### Inferred Development Environments

- **Claude Code** (identified by `Co-Authored-By: Claude <Model> <noreply@anthropic.com>`
  or `Generated with [Claude Code]` markers): ~16 commits
- **Cursor IDE** (identified by `cursoragent@cursor.com` or `Cursor Auto`/`Cursor AI`):
  ~10 commits
- **Generic Claude** (plain `Assisted-by: Claude` — could be API, Claude.ai chat, or
  Claude Code without auto-trailers): ~50 commits

### Claude Model Variants Referenced

| Model | Count |
|-------|-------|
| Claude Sonnet 4/4.5 | 12 |
| Claude Opus 4.5 | 5 |
| Claude Opus 4.5 (slug format: `claude-4.5-opus`) | 4 |
| Claude Opus 4.6 | 1 |
| Claude 4.5 (variant unspecified) | 1 |

## Company Breakdown

| Company | AI-attributed commits |
|---------|----------------------|
| Red Hat | 75 |
| Other (gmail.com) | 1 |

98.7% of AI-attributed commits are from Red Hat engineers, which likely reflects
both Red Hat's large contribution share and potentially earlier/broader internal
adoption of the policy.

## Trailer Format Compliance

### Trailer types used

| Format | Count | Notes |
|--------|-------|-------|
| `Assisted-by:` (proper trailer) | 52 | Most common, policy-compliant |
| `Co-authored-by:` (proper trailer) | 18 | Policy-compliant, often auto-added by tooling |
| `Generated with [Claude Code]` (inline text) | 6 | Not a git trailer, informational only |
| `Assited-by:` (typo) | 4 | Misspelling, 1 contributor |
| `Assisted by Cursor` (no colon) | 4 | Not a valid git trailer, missing colon, 1 contributor |

### Case inconsistencies in trailer names

Git trailers are case-insensitive by spec, so these all work — but the
inconsistency suggests there is no project-wide convention enforced.

| Variant | Count |
|---------|-------|
| `Assisted-by:` | 30 |
| `Assisted-By:` | 22 |
| `Co-Authored-By:` | 10 |
| `Co-authored-by:` | 8 |

### Trailer value variants

There are 17 distinct trailer value formats across only 76 commits:

| Trailer Value | Count | Assessment |
|---|---|---|
| `Assisted-By: Claude <noreply@anthropic.com>` | 21 | Matches policy example exactly |
| `Assisted-by: Claude <noreply@anthropic.com>` | 19 | Matches policy (case-insensitive) |
| `Co-authored-by: Cursor <cursoragent@cursor.com>` | 7 | Valid, auto-added by Cursor |
| `Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>` | 5 | Valid, includes model name |
| `Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>` | 4 | Valid, includes model name |
| `Assited-by: Claude Sonnet 4.5 <noreply@anthropic.com>` | 4 | Typo in trailer name |
| `Assisted-by: claude-4.5-opus` | 4 | Missing email, uses model slug |
| `Assisted-by: claude-sonnet-4-5` | 1 | Missing email, uses model slug |
| `Assisted-by: claude-sonnet-4` | 1 | Missing email, uses model slug |
| `Assisted-by: claude-4-sonnet` | 1 | Missing email, uses model slug |
| `Assisted-by: claude-4.5-sonnet` | 1 | Missing email, uses model slug |
| `Assisted-by: claude-4.5` | 1 | Missing email, uses model slug |
| `Assisted-by: Cursor Auto` | 1 | Non-standard value |
| `Assisted-by: Cursor` | 1 | Missing email |
| `Assisted-By: Claude-4-sonnet` | 1 | Missing email, uses model slug |
| `Co-authored-by: Cursor AI <ai@cursor.com>` | 1 | Different email than standard Cursor |
| `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>` | 1 | Valid, includes model name |

## PR Metrics: AI-Attributed vs Non-AI

Data sourced from 35 AI-attributed PRs and 315 non-AI PRs merged during the
release-1.8 cycle via the GitHub API.

### Time to Merge

| Metric | AI PRs | Non-AI PRs |
|--------|--------|------------|
| Median | 7.5 days | 8.9 days |
| Mean | 14.3 days | 22.9 days |
| Min | 0.7 hours | 0.1 hours |
| Max | 65.6 days | 250.9 days |

AI PRs merge slightly faster on median (~1.4 days quicker). The mean difference
is more pronounced (14.3 vs 22.9 days) but this is driven by a longer tail of
slow-moving non-AI PRs rather than a fundamental difference.

#### Size-Bucketed Time to Merge

Controlling for PR size to allow a fairer comparison:

| Size Bucket | AI PRs (count) | AI Median TTM | Non-AI PRs (count) | Non-AI Median TTM |
|-------------|----------------|---------------|--------------------|--------------------|
| Small (<50 lines) | 13 | 4.3 days | 180 | 3.7 days |
| Medium (50-200 lines) | 14 | 8.7 days | 70 | 13.0 days |
| Large (200+ lines) | 8 | 22.3 days | 65 | 19.5 days |

When normalised for size, AI PRs in the medium bucket merge noticeably faster
(8.7 vs 13.0 days). Small and large PRs show no significant difference.

### PR Size

| Metric | AI PRs | Non-AI PRs |
|--------|--------|------------|
| Median additions | 88 | 24 |
| Median deletions | 17 | 11 |
| Median changed files | 5 | 3 |
| Median commits | 1 | 1 |
| Mean additions | 643 | 645 |
| Mean deletions | 73 | 348 |

AI-attributed PRs tend to be larger (median 88 vs 24 additions), which is
consistent with AI tooling being used to tackle larger refactoring and feature
work rather than small fixes.

#### Size Label Distribution (AI PRs)

| Size | Count |
|------|-------|
| XS | 3 |
| S | 2 |
| M | 7 |
| L | 16 |
| XL | 3 |
| XXL | 4 |

65% of AI PRs are size L or larger. 7 of the 35 AI PRs carry the `approved-vep`
label, indicating they implement KubeVirt Enhancement Proposals.

### Review Activity

| Metric | AI PRs | Non-AI PRs |
|--------|--------|------------|
| Median issue comments | 10 | 12 |
| Median review comments | 4 | 2 |
| Median total comments | 19 | 16 |
| Mean total comments | 26.6 | 22.1 |
| Median review submissions | 7 | 4 |
| Mean review submissions | 9.6 | 7.6 |
| Median approvals | 1 | 1 |
| Mean approvals | 1.5 | 1.0 |

AI PRs receive slightly more review comments (median 4 vs 2) and more review
submissions (median 7 vs 4), likely a function of their larger size rather than
an indication of lower quality. They also tend to receive more approvals on
average (1.5 vs 1.0), again consistent with larger, cross-area changes needing
sign-off from multiple reviewers.

#### Review Intensity (Comments per 100 Lines Changed)

| Size Bucket | AI PRs | Non-AI PRs |
|-------------|--------|------------|
| Small (<50 lines) | 35.4 | 100.0 |
| Medium (50-200 lines) | 15.4 | 15.0 |
| Large (200+ lines) | 2.5 | 6.2 |

For small PRs, non-AI changes receive disproportionately more comments per line
(likely because very small PRs often attract discussion relative to their size).
For medium PRs the review intensity is essentially identical. For large PRs,
AI-attributed changes receive fewer comments per line (2.5 vs 6.2), which could
indicate either cleaner code or less thorough review of large AI-generated
changes — this warrants monitoring.

### Changes Requested

| Rounds | AI PRs | Non-AI PRs |
|--------|--------|------------|
| 0 | 32 (91%) | 288 (91%) |
| 1 | 1 (3%) | 19 (6%) |
| 2 | 1 (3%) | 5 (2%) |
| 3+ | 1 (3%) | 3 (1%) |

The rate of PRs receiving formal "changes requested" reviews is identical at 9%
for both AI and non-AI PRs. This suggests AI-attributed contributions are not
generating more reviewer pushback than human-only contributions.

### AI PR Categories

| Category | Count | Examples |
|----------|-------|---------|
| Feature/enhancement | 13 | VEP 165: Containerpath Volumes, Passt Beta core backend, Guest panic events |
| Test improvement | 9 | Replacing gomega polling with watches, moving tests to correct packages |
| Refactoring | 6 | Extracting KVM/PanicDevices/HostDevice logic, operator refactoring |
| Bug fix | 5 | ComponentImages hash fix, linter coverage fixes, PatchStatus propagation |
| Other | 2 | NAD RBAC rules, RestartRequired clearing |

AI tooling is being used across the full spectrum of contribution types, not just
boilerplate or mechanical changes. Notable uses include VEP implementations
(7 PRs carry `approved-vep`), core networking features (Passt), and significant
refactoring of the converter/launcher subsystem.

#### Areas Touched by AI PRs

| Area | PRs |
|------|-----|
| area/launcher | 12 |
| area/controller | 8 |
| area/operator | 7 |
| area/handler | 5 |
| area/instancetype | 5 |
| area/virtctl | 2 |
| area/api-server | 1 |
| area/monitoring | 1 |

## Key Findings

### Policy Compliance

1. **Good adoption**: 15 contributors (predominantly Red Hat) are attributing AI
   across 10.3% of non-merge commits. The policy is being followed.

2. **100% DCO compliance**: Every AI-attributed commit has a `Signed-off-by`,
   meeting the policy requirement.

3. **Format fragmentation is the main issue**: There are 17 distinct trailer
   value formats across only 76 commits. The policy shows
   `Assisted-by: Claude <noreply@anthropic.com>` as the canonical example, but
   contributors vary by:
   - Including model names (`Claude Opus 4.5` vs `Claude`)
   - Using model ID slugs instead of names (`claude-4.5-opus`)
   - Omitting email addresses entirely
   - Misspelling the trailer name (`Assited-by`)
   - Using free-text instead of trailers (`Assisted by Cursor.`)

4. **`Generated-by:` trailer** — defined in the policy — was never used.
   Contributors chose `Assisted-by` and `Co-authored-by` exclusively.

5. **Tooling auto-adds inconsistent formats**: Claude Code adds
   `Co-Authored-By: Claude <Model> <noreply@anthropic.com>` with model name,
   while Cursor adds `Co-authored-by: Cursor <cursoragent@cursor.com>`. These
   are fine but differ from the policy's examples which use just `Claude`
   without model specifics.

### PR Quality and Review Impact

6. **No measurable quality difference**: AI-attributed PRs have the same rate of
   "changes requested" reviews as non-AI PRs (9% each), suggesting reviewers are
   not finding more issues with AI-assisted code.

7. **AI PRs skew larger**: AI-attributed PRs have a median of 88 additions vs 24
   for non-AI, with 65% at size L or above. This indicates AI tooling is being
   used for substantial work — VEP implementations, subsystem refactoring, and
   feature development — not just trivial changes.

8. **Merge times are comparable**: AI PRs merge in a median of 7.5 days vs 8.9
   days for non-AI. When controlled for size, medium PRs with AI attribution
   merge notably faster (8.7 vs 13.0 days).

9. **Review intensity warrants monitoring**: Large AI PRs receive fewer comments
   per line of code (2.5 vs 6.2 per 100 lines). This could reflect cleaner code
   or could indicate that reviewers spend less time per line on large
   AI-generated diffs, which is worth watching.

10. **Broad subsystem coverage**: AI tooling is being used across launcher,
    controller, operator, handler, and instancetype areas. 7 AI PRs implement
    approved VEPs, showing use in significant architectural work.

## Recommendations

### Trailer Format

- **Standardize on 1-2 canonical trailer formats** in the policy and add a
  `git interpret-trailers` alias or commit hook to enforce them.
- **Clarify whether model names/versions should be included** — currently ~30%
  of Claude attributions include the model, ~70% don't.
- **Require email in trailer values** — `Assisted-by: claude-4.5-opus` without
  an email isn't a well-formed git trailer identity.
- **Consider a CI check** (prow/tide) that validates AI attribution trailer
  format on PRs, at minimum catching typos like `Assited-by`.

### Review Process

- **Monitor review depth on large AI PRs**: The lower comment-per-line rate on
  large AI-attributed PRs may not be a problem today but should be tracked over
  time to ensure review thoroughness does not decline as AI-generated diffs grow.
- **Continue tracking these metrics across releases**: This baseline from
  release-1.8 can be compared against future releases to identify trends in AI
  adoption, PR quality, and review patterns.
