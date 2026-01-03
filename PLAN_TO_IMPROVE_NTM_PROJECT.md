# NTM Improvement Plan

> **Document Purpose**: This is a comprehensive, self-contained strategic plan for improving NTM (Neural Terminal Manager). It is designed to be read by any LLM or human without requiring additional context—everything needed to understand and evaluate the plan is included here.

---

## About This Document

This plan outlines strategic improvements to elevate **NTM** from a capable power-user tool to the definitive command center for AI-assisted development. The document covers:

1. **What NTM is** and its role in the broader ecosystem
2. **The complete tool ecosystem** (the "Dicklesworthstone Stack")
3. **Existing integrations** (CAAM, CM, SLB, Agent Mail)
4. **Underexplored high-value integrations** (bv robot modes, CASS search, s2p, UBS)
5. **Concrete implementation patterns** with Go code examples
6. **Priority matrix** for implementation sequencing

**Key Insight**: NTM is the **cockpit** of an Agentic Coding Flywheel—an orchestration layer that coordinates multiple AI coding agents working in parallel. Most of the ecosystem tools have capabilities that remain **largely untapped** by NTM's current implementation.

---

## Table of Contents

1. [What is NTM?](#what-is-ntm)
2. [The Dicklesworthstone Stack (Complete Ecosystem)](#the-dicklesworthstone-stack-complete-ecosystem)
3. [The Agentic Coding Flywheel](#the-agentic-coding-flywheel)
4. [Current Integration Status](#current-integration-status)
5. [UNDEREXPLORED: bv (Beads Viewer) Robot Modes](#underexplored-bv-beads-viewer-robot-modes)
6. [UNDEREXPLORED: CASS Historical Context Injection](#underexplored-cass-historical-context-injection)
7. [UNDEREXPLORED: s2p (Source-to-Prompt) Context Preparation](#underexplored-s2p-source-to-prompt-context-preparation)
8. [UNDEREXPLORED: UBS Dashboard & Agent Notifications](#underexplored-ubs-dashboard--agent-notifications)
9. [Existing Integration: CAAM (Coding Agent Account Manager)](#existing-integration-caam-coding-agent-account-manager)
10. [Existing Integration: CASS Memory System (CM)](#existing-integration-cass-memory-system-cm)
11. [Existing Integration: SLB (Safety Guardrails)](#existing-integration-slb-safety-guardrails)
12. [Existing Integration: MCP Agent Mail](#existing-integration-mcp-agent-mail)
13. [Ecosystem Discovery: Additional Tools](#ecosystem-discovery-additional-tools)
14. [Priority Matrix](#priority-matrix)
15. [Unified Architecture](#unified-architecture)
16. [Web Dashboard](#web-dashboard)
17. [Implementation Roadmap](#implementation-roadmap)
18. [Success Metrics](#success-metrics)

---

## What is NTM?

### Overview

**NTM (Neural Terminal Manager)** is a Go-based command-line tool for orchestrating multiple AI coding agents in parallel within tmux sessions. It allows developers to:

- **Spawn** multiple AI agents (Claude, Codex, Gemini) in parallel tmux panes
- **Monitor** agent status (idle, working, error, waiting for input)
- **Coordinate** work distribution across agents
- **Track** context window usage and trigger rotations
- **Provide** robot-mode JSON output for programmatic consumption

### Core Capabilities

| Capability | Command | Description |
|-----------|---------|-------------|
| **Spawn sessions** | `ntm spawn myproject --cc=3 --cod=2` | Create tmux session with 3 Claude + 2 Codex agents |
| **List sessions** | `ntm list` | Show all active NTM sessions with agent counts |
| **Monitor status** | `ntm status myproject` | Real-time TUI showing all agent states |
| **Robot output** | `ntm --robot-status` | JSON output for programmatic integration |
| **Kill sessions** | `ntm kill myproject` | Terminate session and all agents |
| **Dashboard** | `ntm dashboard` | Web-based monitoring (planned) |

### Agent Types Supported

| Type | CLI | Provider | Strengths |
|------|-----|----------|-----------|
| `cc` | Claude Code | Anthropic | Analysis, architecture, complex refactoring |
| `cod` | Codex CLI | OpenAI | Fast implementations, bug fixes |
| `gmi` | Gemini CLI | Google | Documentation, research, multi-modal |

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        NTM Core                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  CLI Layer   │  │  TUI Layer   │  │ Robot Layer  │          │
│  │  (commands)  │  │  (bubbletea) │  │  (JSON API)  │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                   │
│         └─────────────────┼─────────────────┘                   │
│                           │                                     │
│  ┌────────────────────────▼─────────────────────────────────┐   │
│  │                    Session Manager                        │   │
│  │  - Spawn/kill tmux sessions                               │   │
│  │  - Manage agent panes                                     │   │
│  │  - Track agent state                                      │   │
│  └────────────────────────┬─────────────────────────────────┘   │
│                           │                                     │
│  ┌────────────────────────▼─────────────────────────────────┐   │
│  │                     tmux Backend                          │   │
│  │  - CreateSession, KillSession                             │   │
│  │  - GetPanes, CapturePaneOutput                            │   │
│  │  - SendKeys                                               │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Key Source Files

| File | Purpose |
|------|---------|
| `cmd/ntm/main.go` | CLI entry point, flag parsing |
| `internal/cli/` | Command implementations (spawn, list, kill, status) |
| `internal/robot/` | Robot mode JSON output generators |
| `internal/tmux/` | tmux session/pane management |
| `internal/status/` | Agent state detection (idle, working, error) |
| `internal/monitor/` | Real-time agent monitoring |
| `internal/context/` | Context window tracking |
| `internal/pipeline/` | Multi-stage pipeline execution |

---

## The Dicklesworthstone Stack (Complete Ecosystem)

NTM is part of a larger ecosystem of coordinated tools designed for AI-assisted software development. Understanding this ecosystem is crucial for understanding the integration opportunities.

### Tool Overview

| Tool | Command | Language | LOC | Purpose | Current NTM Integration |
|------|---------|----------|-----|---------|------------------------|
| **NTM** | `ntm` | Go | ~15K | Agent orchestration (this project) | N/A (this is NTM) |
| **MCP Agent Mail** | `am` | Python | ~8K | Inter-agent messaging, file reservations | ✅ Basic |
| **UBS** | `ubs` | Python | ~12K | Static bug scanning (8 languages) | ✅ Via `internal/scanner/` |
| **Beads/bv** | `bd`, `bv` | Go | ~10K | Issue tracking with dependency graphs | ⚠️ Minimal (uses GetReady) |
| **CASS** | `cass` | Rust | ~50K | Session indexing across 11 agent types | ❌ None |
| **CASS Memory (CM)** | `cm` | Python | ~5K | Three-layer cognitive memory | ⚠️ Planned |
| **CAAM** | `caam` | Python | ~3K | Account switching, rate limit failover | ⚠️ Planned |
| **SLB** | `slb` | Go | ~4K | Two-person rule for dangerous commands | ⚠️ Planned |
| **s2p** | `s2p` | TypeScript | ~3.5K | Source-to-prompt conversion with token counting | ❌ None |

### Ecosystem Relationships

```
                    ┌─────────────────────────────────────┐
                    │           Human Developer           │
                    └────────────────┬────────────────────┘
                                     │
                    ┌────────────────▼────────────────────┐
                    │              NTM                     │
                    │   (Central Orchestration Layer)     │
                    └────────────────┬────────────────────┘
                                     │
       ┌─────────────┬───────────────┼───────────────┬─────────────┐
       │             │               │               │             │
       ▼             ▼               ▼               ▼             ▼
┌────────────┐ ┌──────────┐ ┌───────────────┐ ┌──────────┐ ┌────────────┐
│    CAAM    │ │   SLB    │ │  Agent Mail   │ │   bv     │ │    CASS    │
│ (Accounts) │ │ (Safety) │ │ (Messaging)   │ │ (Tasks)  │ │ (History)  │
└─────┬──────┘ └────┬─────┘ └───────┬───────┘ └────┬─────┘ └─────┬──────┘
      │             │               │              │             │
      │             │               │              │             │
      └─────────────┴───────────────┼──────────────┴─────────────┘
                                    │
                    ┌───────────────▼───────────────┐
                    │         AI Agents             │
                    │  Claude | Codex | Gemini      │
                    └───────────────┬───────────────┘
                                    │
       ┌────────────────────────────┼────────────────────────────┐
       │                            │                            │
       ▼                            ▼                            ▼
┌────────────────┐         ┌───────────────┐          ┌─────────────────┐
│      UBS       │         │      CM       │          │       s2p       │
│ (Bug Scanning) │         │   (Memory)    │          │ (Context Prep)  │
└────────────────┘         └───────────────┘          └─────────────────┘
```

### Tool Details

#### 1. UBS (Ultimate Bug Scanner)
- **Purpose**: Multi-language static analysis with 1000+ bug patterns
- **Languages**: Python, JavaScript, TypeScript, Go, Rust, Java, C, C++
- **Output**: JSON/JSONL with severity, line numbers, fix suggestions
- **Performance**: Sub-5-second scans on typical codebases
- **NTM Integration**: Already integrated via `internal/scanner/` with Beads bridge

#### 2. bv (Beads Viewer)
- **Purpose**: Issue/task tracking with dependency graph analysis
- **Key Feature**: **12+ robot modes** for AI-consumable output
- **Graph Analysis**: PageRank, betweenness centrality, HITS, k-core decomposition
- **Execution Planning**: Parallel track generation for multi-agent work distribution
- **NTM Integration**: **UNDEREXPLORED** - Only uses basic `GetReadyPreview`

#### 3. CASS (Coding Agent Session Search)
- **Purpose**: Unified indexing of sessions from 11 agent types
- **Agents Indexed**: Claude, Codex, Cursor, Aider, Roo, Cline, Windsurf, etc.
- **Search**: Full-text, semantic, and hybrid search across all sessions
- **Performance**: Sub-60ms search latency on 100K+ sessions
- **NTM Integration**: **NONE** - Major opportunity for historical context

#### 4. CM (CASS Memory)
- **Purpose**: Three-layer cognitive memory (episodic/working/procedural)
- **Key Feature**: Rules with confidence scoring and 90-day decay
- **Pipeline**: ACE (Acquire, Consolidate, Express) for rule lifecycle
- **NTM Integration**: Planned but not implemented

#### 5. CAAM (Coding Agent Account Manager)
- **Purpose**: Sub-100ms account switching for rate limit failover
- **Key Feature**: Vault-based OAuth token management
- **Smart Rotation**: Multi-factor scoring (health, recency, cooldown)
- **NTM Integration**: Planned but not implemented

#### 6. SLB (Safety Guardrails)
- **Purpose**: Two-person rule for dangerous commands
- **Risk Tiers**: CRITICAL (2+ approvals), DANGEROUS (1), CAUTION (delay), SAFE
- **Key Feature**: HMAC-signed reviews for audit trail
- **NTM Integration**: Planned but not implemented

#### 7. s2p (source_to_prompt_tui)
- **Purpose**: Convert source code to LLM-ready prompts
- **Key Feature**: Real-time token counting to prevent context overflow
- **Output**: Structured XML for reliable LLM parsing
- **NTM Integration**: **NONE** - Opportunity for context preparation

#### 8. MCP Agent Mail
- **Purpose**: Inter-agent messaging and file reservation
- **Key Feature**: Git-backed message persistence
- **File Reservations**: Prevent multi-agent edit conflicts
- **NTM Integration**: Basic integration exists, could be deeper

---

## The Agentic Coding Flywheel

The tools form a closed-loop learning system where each cycle compounds:

```
                    ┌────────────────────────────────────────┐
                    │                                        │
    ┌───────────────▼───────────────┐                        │
    │        PLAN (Beads/bv)        │                        │
    │   - Ready work queue          │                        │
    │   - Dependency graph          │                        │
    │   - Priority scoring          │                        │
    │   - Execution track planning  │ ◀── NEW: Use bv       │
    │   - Graph-based prioritization│     robot modes       │
    └───────────────┬───────────────┘                        │
                    │                                        │
    ┌───────────────▼───────────────┐                        │
    │    COORDINATE (Agent Mail)    │                        │
    │   - File reservations         │                        │
    │   - Message routing           │                        │
    │   - Thread tracking           │                        │
    └───────────────┬───────────────┘                        │
                    │                                        │
    ┌───────────────▼───────────────┐                        │
    │      EXECUTE (NTM + Agents)   │ ◀── SAFETY (SLB)       │
    │   - Multi-agent sessions      │     Two-person rule    │
    │   - Account rotation (CAAM)   │     for dangerous ops  │
    │   - Parallel task dispatch    │                        │
    │   - Context preparation (s2p) │ ◀── NEW                │
    │   - Historical context (CASS) │ ◀── NEW                │
    └───────────────┬───────────────┘                        │
                    │                                        │
    ┌───────────────▼───────────────┐                        │
    │         SCAN (UBS)            │                        │
    │   - Static analysis           │                        │
    │   - Bug detection             │                        │
    │   - Pre-commit checks         │                        │
    │   - Agent notifications       │ ◀── NEW                │
    └───────────────┬───────────────┘                        │
                    │                                        │
    ┌───────────────▼───────────────┐                        │
    │    REMEMBER (CASS + CM)       │                        │
    │   - Session indexing          │                        │
    │   - Rule extraction           │                        │
    │   - Confidence scoring        │                        │
    └───────────────┴────────────────────────────────────────┘
```

---

## Current Integration Status

### Integration Maturity Levels

| Integration | Status | Maturity | Gap Analysis |
|-------------|--------|----------|--------------|
| **UBS** | ✅ Implemented | Production | Dashboard/notifications missing |
| **bv (basic)** | ⚠️ Minimal | PoC | 12+ robot modes unused |
| **Agent Mail** | ✅ Implemented | Basic | File reservations underused |
| **CAAM** | ❌ Planned | Design | Rate limit failover missing |
| **CM** | ❌ Planned | Design | Memory injection missing |
| **SLB** | ❌ Planned | Design | Safety gates missing |
| **CASS** | ❌ None | Gap | Historical context missing |
| **s2p** | ❌ None | Gap | Context preparation missing |

### The Gap: What's Missing

```
┌─────────────────────────────────────────────────────────────────┐
│                   CURRENT STATE                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  NTM spawns agents → Agents work → NTM monitors status          │
│                                                                 │
│  Problems:                                                       │
│  ❌ No smart task distribution (agents get random work)          │
│  ❌ No historical context (agents reinvent solutions)            │
│  ❌ No token budget management (agents overflow context)         │
│  ❌ No rate limit failover (agents hit limits, stop)             │
│  ❌ No safety gates (dangerous commands execute unchecked)       │
│  ❌ No bug notifications (agents don't know about scan results) │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                   TARGET STATE                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  NTM spawns agents with:                                         │
│  ✅ Smart task assignment (bv graph analysis → optimal pairing)  │
│  ✅ Historical context (CASS → relevant past solutions)          │
│  ✅ Token budgets (s2p → context fits in window)                 │
│  ✅ Automatic failover (CAAM → seamless account switching)       │
│  ✅ Safety gates (SLB → dangerous commands require approval)     │
│  ✅ Bug notifications (UBS → agents notified of issues)          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## UNDEREXPLORED: bv (Beads Viewer) Robot Modes

### The Opportunity

The `bv` command has **12+ robot modes** designed for AI consumption that NTM barely uses. These modes provide sophisticated graph analysis, execution planning, and work distribution that could dramatically improve agent coordination.

### Available Robot Modes

| Mode | Command | Output | NTM Usage |
|------|---------|--------|-----------|
| `--robot-status` | `bv --robot-status` | Project health summary | ❌ Unused |
| `--robot-ready` | `bv --robot-ready` | Ready work with priorities | ✅ Used |
| `--robot-blocked` | `bv --robot-blocked` | Blocked items with blockers | ❌ Unused |
| `--robot-graph` | `bv --robot-graph` | Full dependency graph | ❌ Unused |
| `--robot-insights` | `bv --robot-insights` | Graph metrics (PageRank, etc.) | ❌ Unused |
| `--robot-plan` | `bv --robot-plan` | Execution plan with tracks | ❌ Unused |
| `--robot-triage` | `bv --robot-triage` | Prioritized triage list | ❌ Unused |
| `--robot-triage-by-track` | `bv --robot-triage-by-track` | Work grouped by track | ❌ Unused |
| `--robot-triage-by-label` | `bv --robot-triage-by-label` | Work grouped by label | ❌ Unused |
| `--robot-label-health` | `bv --robot-label-health` | Label usage analysis | ❌ Unused |
| `--robot-suggest` | `bv --robot-suggest` | Suggested dependencies | ❌ Unused |
| `--robot-feedback` | `bv --robot-feedback` | Learning loop data | ❌ Unused |

### Integration 1: Track-Based Agent Assignment

Instead of random assignment, use bv's execution plan to assign entire **tracks** to agents:

```go
// internal/robot/assign.go - ENHANCED

type ExecutionTrack struct {
    TrackID   string   `json:"track_id"`
    Items     []string `json:"items"`      // Bead IDs in dependency order
    Parallel  bool     `json:"parallel"`   // Can execute in parallel with other tracks
    EstEffort int      `json:"est_effort"` // Estimated effort units
}

type ExecutionPlan struct {
    Tracks     []ExecutionTrack `json:"tracks"`
    CritPath   []string         `json:"critical_path"`
    Parallelism int             `json:"max_parallelism"`
}

// getExecutionPlan fetches bv's execution plan for the project
func getExecutionPlan() (*ExecutionPlan, error) {
    out, err := exec.Command("bv", "--robot-plan", "--json").Output()
    if err != nil {
        return nil, fmt.Errorf("bv --robot-plan failed: %w", err)
    }

    var plan ExecutionPlan
    if err := json.Unmarshal(out, &plan); err != nil {
        return nil, err
    }
    return &plan, nil
}

// assignByTracks distributes work to agents by track, not individual items
func assignByTracks(agents []assignAgentInfo, plan *ExecutionPlan) []AssignRecommend {
    var recommendations []AssignRecommend

    idleAgents := filterIdle(agents)

    // Sort tracks by: critical path first, then by effort (larger first)
    sortedTracks := sortTracks(plan.Tracks, plan.CritPath)

    for i, track := range sortedTracks {
        if i >= len(idleAgents) {
            break
        }

        agent := idleAgents[i]

        // Assign entire track to one agent for coherent work
        recommendations = append(recommendations, AssignRecommend{
            Agent:      fmt.Sprintf("%d", agent.paneIdx),
            AgentType:  agent.agentType,
            Model:      agent.model,
            AssignBead: track.Items[0], // Start with first item
            BeadTitle:  fmt.Sprintf("Track %s (%d items)", track.TrackID, len(track.Items)),
            Priority:   determinePriority(track, plan.CritPath),
            Confidence: 0.9, // High confidence - bv planned this
            Reasoning:  fmt.Sprintf("Track-based assignment: %d related items in dependency order", len(track.Items)),
            TrackItems: track.Items, // NEW: All items in track
        })
    }

    return recommendations
}
```

### Integration 2: Graph-Based Prioritization

Use bv's graph metrics to identify the most impactful work:

```go
// internal/robot/insights.go - NEW FILE

type GraphInsights struct {
    // PageRank: Items that many others depend on (high = critical)
    PageRank map[string]float64 `json:"pagerank"`

    // Betweenness: Items that connect different parts of the graph (high = bridges)
    Betweenness map[string]float64 `json:"betweenness"`

    // InDegree: How many items depend on this (high = blocking many)
    InDegree map[string]int `json:"in_degree"`

    // KCore: Core decomposition (high = deeply embedded)
    KCore map[string]int `json:"k_core"`
}

// getGraphInsights fetches bv's graph analysis
func getGraphInsights() (*GraphInsights, error) {
    out, err := exec.Command("bv", "--robot-insights", "--json").Output()
    if err != nil {
        return nil, err
    }

    var insights GraphInsights
    if err := json.Unmarshal(out, &insights); err != nil {
        return nil, err
    }
    return &insights, nil
}

// prioritizeByImpact uses graph metrics to identify highest-impact work
func prioritizeByImpact(beads []bv.BeadPreview, insights *GraphInsights) []bv.BeadPreview {
    // Score each bead by composite metric
    type scored struct {
        bead  bv.BeadPreview
        score float64
    }

    var scoredBeads []scored
    for _, b := range beads {
        score := 0.0

        // High PageRank = many things depend on this
        if pr, ok := insights.PageRank[b.ID]; ok {
            score += pr * 100
        }

        // High InDegree = blocking many items
        if deg, ok := insights.InDegree[b.ID]; ok {
            score += float64(deg) * 10
        }

        // High Betweenness = bridge between clusters
        if bc, ok := insights.Betweenness[b.ID]; ok {
            score += bc * 50
        }

        scoredBeads = append(scoredBeads, scored{b, score})
    }

    // Sort by score descending
    sort.Slice(scoredBeads, func(i, j int) bool {
        return scoredBeads[i].score > scoredBeads[j].score
    })

    result := make([]bv.BeadPreview, len(scoredBeads))
    for i, s := range scoredBeads {
        result[i] = s.bead
    }
    return result
}
```

### Integration 3: Smart Suggestions for Dependency Discovery

Use bv's suggestion system to discover implicit dependencies:

```go
// internal/robot/suggest.go - NEW FILE

type DependencySuggestion struct {
    FromID     string  `json:"from_id"`
    ToID       string  `json:"to_id"`
    Confidence float64 `json:"confidence"`
    Reason     string  `json:"reason"`
}

// getSuggestions fetches bv's dependency suggestions
func getSuggestions() ([]DependencySuggestion, error) {
    out, err := exec.Command("bv", "--robot-suggest", "--json").Output()
    if err != nil {
        return nil, err
    }

    var suggestions []DependencySuggestion
    if err := json.Unmarshal(out, &suggestions); err != nil {
        return nil, err
    }
    return suggestions, nil
}

// applyHighConfidenceSuggestions auto-applies suggestions above threshold
func applyHighConfidenceSuggestions(threshold float64) error {
    suggestions, err := getSuggestions()
    if err != nil {
        return err
    }

    for _, s := range suggestions {
        if s.Confidence >= threshold {
            // Apply the dependency
            cmd := exec.Command("bd", "dep", "add", s.FromID, s.ToID)
            if err := cmd.Run(); err != nil {
                log.Printf("Failed to add dependency %s -> %s: %v", s.FromID, s.ToID, err)
            }
        }
    }
    return nil
}
```

### Integration 4: Label-Based Work Distribution

Use bv's label analysis for specialized agent assignment:

```go
// internal/robot/labels.go - NEW FILE

type LabelHealth struct {
    Label       string   `json:"label"`
    OpenCount   int      `json:"open_count"`
    ClosedCount int      `json:"closed_count"`
    StaleCount  int      `json:"stale_count"`
    RecentItems []string `json:"recent_items"`
}

// getLabelHealth fetches bv's label analysis
func getLabelHealth() ([]LabelHealth, error) {
    out, err := exec.Command("bv", "--robot-label-health", "--json").Output()
    if err != nil {
        return nil, err
    }

    var health []LabelHealth
    if err := json.Unmarshal(out, &health); err != nil {
        return nil, err
    }
    return health, nil
}

// assignByLabel matches labels to specialized agents
func assignByLabel(agents []assignAgentInfo) ([]AssignRecommend, error) {
    // Get work grouped by label
    out, err := exec.Command("bv", "--robot-triage-by-label", "--json").Output()
    if err != nil {
        return nil, err
    }

    var triageByLabel map[string][]bv.BeadPreview
    if err := json.Unmarshal(out, &triageByLabel); err != nil {
        return nil, err
    }

    var recommendations []AssignRecommend

    // Label-to-agent-type mapping
    labelAgentMap := map[string]string{
        "security":      "claude", // Claude better at security analysis
        "performance":   "claude", // Claude better at architecture
        "documentation": "gemini", // Gemini good at docs
        "bug":           "codex",  // Codex fast at fixes
        "test":          "codex",  // Codex good at tests
    }

    for label, beads := range triageByLabel {
        preferredType := labelAgentMap[label]

        // Find idle agent of preferred type
        for _, agent := range agents {
            if agent.state == "idle" && agent.agentType == preferredType {
                if len(beads) > 0 {
                    recommendations = append(recommendations, AssignRecommend{
                        Agent:      fmt.Sprintf("%d", agent.paneIdx),
                        AgentType:  agent.agentType,
                        AssignBead: beads[0].ID,
                        BeadTitle:  beads[0].Title,
                        Confidence: 0.85,
                        Reasoning:  fmt.Sprintf("Label '%s' matches %s agent specialty", label, preferredType),
                    })
                    break
                }
            }
        }
    }

    return recommendations, nil
}
```

### New NTM Commands for bv Integration

```bash
# Graph-based work analysis
ntm work insights                     # Show graph metrics for open work
ntm work critical-path                # Show items on critical path
ntm work tracks                       # Show execution tracks

# Smart assignment
ntm assign --strategy=track           # Assign by execution track
ntm assign --strategy=impact          # Assign by graph impact
ntm assign --strategy=label           # Assign by label specialization

# Dependency discovery
ntm work suggest                      # Show suggested dependencies
ntm work suggest --apply=0.8          # Auto-apply high-confidence suggestions

# Robot mode
ntm --robot-tracks                    # JSON execution tracks
ntm --robot-insights                  # JSON graph metrics
```

---

## UNDEREXPLORED: CASS Historical Context Injection

### The Opportunity

CASS indexes **50K+ sessions** across **11 different agent types** with sub-60ms search. NTM could inject relevant historical context before spawning agents, so they don't reinvent solutions.

### CASS Capabilities

| Feature | Description |
|---------|-------------|
| **Multi-agent indexing** | Claude, Codex, Cursor, Aider, Roo, Cline, Windsurf, etc. |
| **Full-text search** | Search across all session content |
| **Semantic search** | Embedding-based similarity search |
| **Hybrid search** | Combined full-text + semantic |
| **Multi-machine** | Unified index across multiple development machines |

### Integration 1: Pre-Task Context Enrichment

Before sending a prompt to an agent, search CASS for relevant past solutions:

```go
// internal/context/historical.go - NEW FILE

type HistoricalContext struct {
    Sessions []SessionSnippet `json:"sessions"`
    Query    string           `json:"query"`
    Count    int              `json:"count"`
}

type SessionSnippet struct {
    SessionID   string    `json:"session_id"`
    AgentType   string    `json:"agent_type"`
    Timestamp   time.Time `json:"timestamp"`
    Snippet     string    `json:"snippet"`
    Relevance   float64   `json:"relevance"`
    ProjectPath string    `json:"project_path,omitempty"`
}

// searchHistoricalContext searches CASS for relevant past sessions
func searchHistoricalContext(task string, limit int) (*HistoricalContext, error) {
    // Use CASS semantic search for best results
    cmd := exec.Command("cass", "search",
        "--query", task,
        "--limit", fmt.Sprintf("%d", limit),
        "--mode", "hybrid",  // Combined full-text + semantic
        "--json",
    )

    out, err := cmd.Output()
    if err != nil {
        // CASS not available, continue without historical context
        log.Printf("CASS search failed (continuing without history): %v", err)
        return &HistoricalContext{Query: task}, nil
    }

    var ctx HistoricalContext
    if err := json.Unmarshal(out, &ctx); err != nil {
        return nil, err
    }
    ctx.Query = task
    return &ctx, nil
}

// formatHistoricalPrompt formats historical context for injection into agent prompt
func formatHistoricalPrompt(ctx *HistoricalContext) string {
    if len(ctx.Sessions) == 0 {
        return ""
    }

    var sb strings.Builder
    sb.WriteString("## Historical Context (from past sessions)\n\n")
    sb.WriteString("Similar problems have been solved before. Here are relevant snippets:\n\n")

    for i, s := range ctx.Sessions {
        sb.WriteString(fmt.Sprintf("### Session %d (%s, %s)\n",
            i+1, s.AgentType, s.Timestamp.Format("2006-01-02")))
        sb.WriteString("```\n")
        // Truncate long snippets
        snippet := s.Snippet
        if len(snippet) > 500 {
            snippet = snippet[:500] + "..."
        }
        sb.WriteString(snippet)
        sb.WriteString("\n```\n\n")
    }

    sb.WriteString("Use these as reference, but adapt to current context.\n\n")
    return sb.String()
}

// enrichPromptWithHistory adds historical context to a task prompt
func enrichPromptWithHistory(prompt string, historyLimit int) (string, error) {
    ctx, err := searchHistoricalContext(prompt, historyLimit)
    if err != nil {
        return prompt, err
    }

    historicalSection := formatHistoricalPrompt(ctx)
    if historicalSection == "" {
        return prompt, nil
    }

    return fmt.Sprintf("%s\n---\n\n%s", historicalSection, prompt), nil
}
```

### Integration 2: Project-Specific History

Search only sessions from the current project:

```go
// internal/context/project_history.go

// searchProjectHistory searches CASS for sessions in the current project
func searchProjectHistory(projectPath, task string, limit int) (*HistoricalContext, error) {
    cmd := exec.Command("cass", "search",
        "--query", task,
        "--project", projectPath,  // Filter to current project
        "--limit", fmt.Sprintf("%d", limit),
        "--mode", "semantic",
        "--json",
    )

    out, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    var ctx HistoricalContext
    if err := json.Unmarshal(out, &ctx); err != nil {
        return nil, err
    }
    return &ctx, nil
}
```

### Integration 3: Error Resolution History

When an agent hits an error, search for past solutions:

```go
// internal/monitor/error_history.go

// searchErrorHistory searches CASS for past solutions to similar errors
func searchErrorHistory(errorMessage string) ([]SessionSnippet, error) {
    // Extract key error terms
    keywords := extractErrorKeywords(errorMessage)
    query := fmt.Sprintf("error fix solution: %s", strings.Join(keywords, " "))

    cmd := exec.Command("cass", "search",
        "--query", query,
        "--limit", "3",
        "--mode", "semantic",
        "--json",
    )

    out, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    var result struct {
        Sessions []SessionSnippet `json:"sessions"`
    }
    if err := json.Unmarshal(out, &result); err != nil {
        return nil, err
    }
    return result.Sessions, nil
}

// extractErrorKeywords extracts searchable terms from error message
func extractErrorKeywords(error string) []string {
    // Remove noise, extract key terms
    // Example: "undefined: foo.Bar" -> ["undefined", "foo.Bar"]
    var keywords []string

    // Split on common separators
    parts := strings.FieldsFunc(error, func(r rune) bool {
        return r == ':' || r == '\n' || r == '\t'
    })

    for _, p := range parts {
        p = strings.TrimSpace(p)
        if len(p) > 3 && len(p) < 100 {
            keywords = append(keywords, p)
        }
    }

    return keywords
}
```

### New NTM Commands for CASS Integration

```bash
# Historical search
ntm history search "implement JWT auth"     # Search past sessions
ntm history project                         # Search current project only
ntm history errors "undefined: foo.Bar"     # Search error solutions

# Context injection
ntm spawn myproject --cc=2 --history=5      # Inject top 5 relevant sessions

# Robot mode
ntm --robot-history "query"                 # JSON historical context
```

---

## UNDEREXPLORED: s2p (Source-to-Prompt) Context Preparation

### The Opportunity

s2p converts source code to LLM-ready prompts with **real-time token counting**. This prevents context overflow—a common problem when agents try to read too many files.

### s2p Capabilities

| Feature | Description |
|---------|-------------|
| **Token counting** | Real-time counting with multiple tokenizer support |
| **Structured output** | XML format for reliable LLM parsing |
| **File selection** | Glob patterns, presets, or interactive TUI |
| **Tree inclusion** | Optional directory structure context |
| **Smart truncation** | Truncates at logical boundaries (functions, classes) |

### Integration 1: Token-Budgeted Context Preparation

Before spawning an agent, prepare context within token budget:

```go
// internal/context/s2p.go - NEW FILE

type S2PConfig struct {
    Files       []string `json:"files"`
    TokenBudget int      `json:"token_budget"`
    IncludeTree bool     `json:"include_tree"`
    Format      string   `json:"format"` // "xml", "markdown", "plain"
}

type S2POutput struct {
    Content     string `json:"content"`
    TokenCount  int    `json:"token_count"`
    FileCount   int    `json:"file_count"`
    Truncated   bool   `json:"truncated"`
    TruncatedAt string `json:"truncated_at,omitempty"`
}

// prepareContext uses s2p to prepare context within token budget
func prepareContext(cfg S2PConfig) (*S2POutput, error) {
    args := []string{
        "--files", strings.Join(cfg.Files, ","),
        "--budget", fmt.Sprintf("%d", cfg.TokenBudget),
        "--format", cfg.Format,
        "--json",
    }

    if cfg.IncludeTree {
        args = append(args, "--tree")
    }

    cmd := exec.Command("s2p", args...)
    out, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("s2p failed: %w", err)
    }

    var output S2POutput
    if err := json.Unmarshal(out, &output); err != nil {
        return nil, err
    }
    return &output, nil
}

// prepareAgentContext prepares context for an agent with budget enforcement
func prepareAgentContext(files []string, agentType string) (*S2POutput, error) {
    // Different agents have different context limits
    budgets := map[string]int{
        "claude": 180000, // Claude has large context
        "codex":  120000, // Codex medium
        "gemini": 100000, // Gemini
    }

    budget := budgets[agentType]
    if budget == 0 {
        budget = 100000 // Default
    }

    return prepareContext(S2PConfig{
        Files:       files,
        TokenBudget: budget,
        IncludeTree: true,
        Format:      "xml", // Best for reliable parsing
    })
}
```

### Integration 2: Automatic File Discovery

Use s2p with glob patterns for automatic file discovery:

```go
// internal/context/discovery.go

// discoverRelevantFiles finds files relevant to a task
func discoverRelevantFiles(task string, projectPath string) ([]string, error) {
    // Use s2p's smart discovery
    cmd := exec.Command("s2p",
        "--discover",
        "--query", task,
        "--path", projectPath,
        "--json",
    )

    out, err := cmd.Output()
    if err != nil {
        // Fallback to basic glob
        return globFiles(projectPath, "**/*.go", "**/*.py", "**/*.js")
    }

    var result struct {
        Files []string `json:"files"`
    }
    if err := json.Unmarshal(out, &result); err != nil {
        return nil, err
    }
    return result.Files, nil
}
```

### Integration 3: Pipeline Stage Context

Prepare context for multi-stage pipelines:

```go
// internal/pipeline/context.go

// preparePipelineContext prepares context for each pipeline stage
func preparePipelineContext(pipeline Pipeline) error {
    for i, stage := range pipeline.Stages {
        // Determine files relevant to this stage
        files, err := discoverRelevantFiles(stage.Prompt, pipeline.WorkDir)
        if err != nil {
            return err
        }

        // Prepare context within budget
        ctx, err := prepareAgentContext(files, stage.AgentType)
        if err != nil {
            return err
        }

        // Inject into stage prompt
        pipeline.Stages[i].Prompt = fmt.Sprintf(
            "## Context (%d tokens, %d files)\n\n%s\n\n---\n\n## Task\n\n%s",
            ctx.TokenCount, ctx.FileCount, ctx.Content, stage.Prompt,
        )

        if ctx.Truncated {
            log.Printf("Stage %d context truncated at %s", i, ctx.TruncatedAt)
        }
    }
    return nil
}
```

### New NTM Commands for s2p Integration

```bash
# Context preparation
ntm context prepare "*.go" --budget=100000   # Prepare Go files
ntm context discover "implement auth"        # Find relevant files

# Spawn with prepared context
ntm spawn myproject --cc=2 --context="internal/**/*.go"

# Robot mode
ntm --robot-context "*.go" --budget=50000    # JSON context output
```

---

## UNDEREXPLORED: UBS Dashboard & Agent Notifications

### The Opportunity

UBS is already integrated in `internal/scanner/`, but the **dashboard integration** and **agent notification** capabilities are minimal. Agents should know about bug scan results.

### Current UBS Integration

The existing integration (`internal/scanner/`) provides:
- Scanning files on demand
- Beads bridge (creating issues from findings)
- JSON output parsing

### What's Missing

1. **Dashboard widget** showing scan status
2. **Agent notifications** when bugs are found
3. **Session-level quality metrics**
4. **Pre-commit integration** with agent awareness

### Integration 1: Dashboard Bug Widget

```go
// internal/dashboard/ubs.go - NEW FILE

type UBSStatus struct {
    LastScan       time.Time       `json:"last_scan"`
    TotalFindings  int             `json:"total_findings"`
    BySeverity     map[string]int  `json:"by_severity"`
    RecentFindings []UBSFinding    `json:"recent_findings"`
    ScanDuration   time.Duration   `json:"scan_duration"`
}

type UBSFinding struct {
    File     string `json:"file"`
    Line     int    `json:"line"`
    Severity string `json:"severity"`
    Message  string `json:"message"`
    Rule     string `json:"rule"`
}

// getUBSStatus fetches current bug scan status
func getUBSStatus() (*UBSStatus, error) {
    // Check for recent scan results
    resultsPath := filepath.Join(".ubs", "last_scan.json")
    data, err := os.ReadFile(resultsPath)
    if err != nil {
        return &UBSStatus{}, nil // No recent scan
    }

    var status UBSStatus
    if err := json.Unmarshal(data, &status); err != nil {
        return nil, err
    }
    return &status, nil
}
```

### Integration 2: Agent Bug Notifications

Notify agents when UBS finds bugs in files they're working on:

```go
// internal/monitor/ubs_notify.go - NEW FILE

type BugNotifier struct {
    session     string
    watchedDirs []string
}

// watchForBugs monitors file changes and notifies agents of new bugs
func (n *BugNotifier) watchForBugs(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    var lastScanTime time.Time

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Check for file changes since last scan
            changedFiles := getChangedFiles(lastScanTime)
            if len(changedFiles) == 0 {
                continue
            }

            // Run quick UBS scan on changed files
            findings := runQuickScan(changedFiles)
            lastScanTime = time.Now()

            if len(findings) > 0 {
                // Notify agents working on affected files
                n.notifyAgents(findings)
            }
        }
    }
}

// notifyAgents sends bug findings to relevant agents via Agent Mail
func (n *BugNotifier) notifyAgents(findings []UBSFinding) {
    // Group findings by file
    byFile := make(map[string][]UBSFinding)
    for _, f := range findings {
        byFile[f.File] = append(byFile[f.File], f)
    }

    // Find which agent is working on each file
    panes, _ := tmux.GetPanes(n.session)
    for _, pane := range panes {
        agentFiles := detectAgentWorkingFiles(pane.ID)

        for file, fileFindings := range byFile {
            if contains(agentFiles, file) {
                // Send notification to this agent
                sendBugNotification(pane, file, fileFindings)
            }
        }
    }
}

// sendBugNotification formats and sends bug notification
func sendBugNotification(pane tmux.Pane, file string, findings []UBSFinding) {
    var msg strings.Builder
    msg.WriteString(fmt.Sprintf("⚠️ UBS found %d issue(s) in %s:\n\n", len(findings), file))

    for _, f := range findings {
        msg.WriteString(fmt.Sprintf("- Line %d [%s]: %s\n", f.Line, f.Severity, f.Message))
    }

    // Send via tmux (appears in agent's pane)
    tmux.SendKeys(pane.ID, fmt.Sprintf("echo '%s'", msg.String()), true)

    // Also send via Agent Mail for persistence
    agentmail.SendMessage(agentmail.Message{
        To:         []string{pane.AgentName},
        Subject:    fmt.Sprintf("Bug findings in %s", file),
        BodyMD:     msg.String(),
        Importance: determineBugImportance(findings),
    })
}

// determineBugImportance sets importance based on severity
func determineBugImportance(findings []UBSFinding) string {
    for _, f := range findings {
        if f.Severity == "critical" || f.Severity == "high" {
            return "high"
        }
    }
    return "normal"
}
```

### Integration 3: Session Quality Metrics

Track bugs introduced/fixed per agent:

```go
// internal/metrics/quality.go - NEW FILE

type SessionQuality struct {
    SessionID      string         `json:"session_id"`
    StartTime      time.Time      `json:"start_time"`
    BugsAtStart    int            `json:"bugs_at_start"`
    BugsNow        int            `json:"bugs_now"`
    BugsFixed      int            `json:"bugs_fixed"`
    BugsIntroduced int            `json:"bugs_introduced"`
    ByAgent        map[string]int `json:"by_agent"` // agent -> net bugs
}

// trackSessionQuality monitors bug count changes
func trackSessionQuality(session string) *SessionQuality {
    sq := &SessionQuality{
        SessionID: session,
        StartTime: time.Now(),
        ByAgent:   make(map[string]int),
    }

    // Get initial bug count
    initialFindings := runFullScan()
    sq.BugsAtStart = len(initialFindings)

    return sq
}

// updateQualityMetrics updates metrics after file changes
func (sq *SessionQuality) updateQualityMetrics(agentName string, changedFiles []string) {
    beforeCount := sq.BugsNow

    // Rescan changed files
    findings := runQuickScan(changedFiles)
    sq.BugsNow = len(findings)

    // Calculate delta
    delta := sq.BugsNow - beforeCount

    if delta > 0 {
        sq.BugsIntroduced += delta
        sq.ByAgent[agentName] += delta
    } else if delta < 0 {
        sq.BugsFixed += -delta
        sq.ByAgent[agentName] += delta // negative = good
    }
}
```

### New NTM Commands for UBS Integration

```bash
# Bug scanning
ntm scan                              # Run UBS scan
ntm scan --watch                      # Continuous scanning
ntm scan --notify                     # Scan with agent notifications

# Quality metrics
ntm quality                           # Show session quality metrics
ntm quality --by-agent                # Quality breakdown by agent

# Robot mode
ntm --robot-bugs                      # JSON bug status
ntm --robot-quality                   # JSON quality metrics
```

---

## Existing Integration: CAAM (Coding Agent Account Manager)

*[This section describes the existing planned integration. Key points:]*

### What CAAM Does

- **Sub-100ms account switching** for AI CLIs
- **Vault-based profiles**: Backup/restore auth without browser OAuth
- **Smart rotation**: Multi-factor scoring for profile selection
- **Rate limit failover**: Automatic account switching on rate limit
- **Health scoring**: Token validity tracking with expiry warnings

### Integration Points

1. **Account assignment on spawn**: `ntm spawn --cc=3 --account-strategy=round-robin`
2. **Automatic rate limit failover**: Detect rate limit, switch account, retry
3. **Health dashboard integration**: Show account status in web dashboard
4. **Robot mode output**: `ntm --robot-accounts`

### Implementation Status: Planned

```go
// internal/accounts/caam.go - PLANNED
func assignAccounts(agentType string, count int, strategy string) ([]string, error) {
    // Query caam for available profiles
    // Distribute agents across profiles
    // Return account assignments
}

func handleRateLimit(paneID, currentProfile string) error {
    // 1. Mark cooldown via caam
    // 2. Get next profile
    // 3. Activate new profile
    // 4. Notify via Agent Mail
}
```

---

## Existing Integration: CASS Memory System (CM)

*[This section describes the existing planned integration. Key points:]*

### What CM Does

- **Three-layer cognitive memory**: Episodic (CASS), Working (Diary), Procedural (Playbook)
- **Cross-agent learning**: Rules from Claude benefit Codex
- **Confidence decay**: 90-day half-life prevents stale rules
- **Evidence-based validation**: Rules require historical support

### Integration Points

1. **Pre-task context injection**: `cm context "task" --json`
2. **Automatic post-session reflection**: `cm reflect --days 1`
3. **Memory-aware task assignment**: Use rules to inform agent selection
4. **Robot mode output**: `ntm --robot-memory`

### Implementation Status: Planned

```go
// internal/memory/context.go - PLANNED
func GetMemoryContext(taskDescription string) (*MemoryContext, error) {
    // Query cm context
    // Parse rules and anti-patterns
    // Format for injection
}

func OnSessionEnd(session string) {
    // Trigger cm reflect
    // Notify if new rules learned
}
```

---

## Existing Integration: SLB (Safety Guardrails)

*[This section describes the existing planned integration. Key points:]*

### What SLB Does

- **Two-person rule** for destructive commands
- **Risk tiers**: CRITICAL (2+), DANGEROUS (1), CAUTION (delay), SAFE
- **HMAC-signed reviews**: Cryptographic audit trail
- **Client-side execution**: Commands run in caller's environment

### Integration Points

1. **SLB-wrapped command execution**: Agents route dangerous commands through SLB
2. **NTM as reviewer dispatcher**: Route approval requests to appropriate agents
3. **Dashboard integration**: Show pending approvals
4. **Emergency override**: `ntm emergency "cmd" --reason="..."`

### Implementation Status: Planned

```go
// internal/slb/executor.go - PLANNED
func (e *CommandExecutor) executeWithSLB(cmd, justification string) error {
    // Use slb run for atomic request+wait+execute
    // Block until approved, rejected, or timeout
}

func WatchSLBRequests(session string) {
    // Watch for pending requests
    // Dispatch to appropriate reviewers
}
```

---

## Existing Integration: MCP Agent Mail

*[This section describes the existing integration. Key points:]*

### What Agent Mail Does

- **Inter-agent messaging**: Markdown messages between agents
- **File reservations**: Prevent multi-agent edit conflicts
- **Thread tracking**: Conversation continuity
- **Git-backed persistence**: All messages versioned

### Current Integration

- Basic session threads
- Basic message routing

### Enhanced Integration Points

1. **Automatic session threads**: Create thread on session spawn
2. **File reservation for parallel agents**: Reserve files before work
3. **Cross-agent communication**: `ntm mail send`, `ntm mail broadcast`

### Implementation Status: Basic (could be deeper)

---

## Ecosystem Discovery: Additional Tools

Research identified **21 total projects** in the ecosystem. Beyond the core 8, these may offer integration value:

### Potentially Valuable

| Tool | Purpose | Integration Opportunity |
|------|---------|------------------------|
| **misc_coding_agent_tips_and_scripts** | Battle-tested patterns | Best practices for NTM workflows |
| **chat_shared_conversation_to_file** | Conversation export | Post-mortem analysis of sessions |
| **automated_lint_testing_scripts** | Lint automation | Pre-commit hooks for agents |

### Lower Priority

| Tool | Purpose | Notes |
|------|---------|-------|
| llm_price_arena | LLM cost comparison | Informational only |
| project_to_jsonl | Project conversion | One-time use |
| repo_to_llm_prompt | Similar to s2p | s2p is more capable |

---

## Priority Matrix

### Effort vs. Impact Analysis

```
                        HIGH IMPACT
                             │
     bv Robot Modes ●────────│──────────● CASS Historical Context
     (Tier 1: Low Effort)    │            (Tier 1: Medium Effort)
                             │
                             │
                             │
LOW ─────────────────────────┼───────────────────────────────── HIGH
EFFORT                       │                                  EFFORT
                             │
                             │
     UBS Notifications ●─────│──────────● s2p Context Prep
     (Tier 2: Low Effort)    │            (Tier 2: Medium Effort)
                             │
                             │
                        LOW IMPACT
```

### Implementation Tiers

#### Tier 1: Immediate High-Value (Do First)

| Integration | Effort | Impact | Why |
|-------------|--------|--------|-----|
| **bv Robot Modes** | Low | High | Already have bv, just use more modes |
| **CASS Historical** | Medium | High | Agents stop reinventing solutions |

#### Tier 2: Strategic (Do Next)

| Integration | Effort | Impact | Why |
|-------------|--------|--------|-----|
| **s2p Context** | Medium | Medium | Prevents context overflow |
| **UBS Notifications** | Low | Medium | Already have UBS, just add notifications |

#### Tier 3: Complete the Ecosystem (Do Later)

| Integration | Effort | Impact | Why |
|-------------|--------|--------|-----|
| **CAAM** | Medium | Medium | Rate limits are annoying but not critical |
| **CM Memory** | High | Medium | Complex, requires pipeline changes |
| **SLB** | Medium | Medium | Safety important but not daily |

### Recommended Sequence

1. **Week 1**: bv robot modes integration (track-based assignment, graph insights)
2. **Week 2**: CASS historical context injection
3. **Week 3**: UBS notifications + dashboard widget
4. **Week 4**: s2p context preparation
5. **Month 2**: CAAM, CM, SLB integrations

---

## Unified Architecture

### The NTM Daemon

To coordinate all integrations, NTM should run a lightweight daemon:

```go
// internal/daemon/daemon.go
type Daemon struct {
    // Existing
    sessionManager *session.Manager

    // Integrations
    accountWatcher  *caam.HealthWatcher    // CAAM
    slbDispatcher   *slb.RequestDispatcher // SLB
    memoryReflector *cm.Reflector          // CM
    mailRouter      *agentmail.Router      // Agent Mail
    bugNotifier     *ubs.Notifier          // UBS (NEW)
    contextPreparer *s2p.Preparer          // s2p (NEW)
    trackPlanner    *bv.TrackPlanner       // bv (NEW)
    historySearcher *cass.Searcher         // CASS (NEW)

    // Web
    webServer *dashboard.Server
}

func (d *Daemon) Start() error {
    // Start all subsystems
    go d.accountWatcher.Watch()
    go d.slbDispatcher.Watch()
    go d.memoryReflector.Watch()
    go d.mailRouter.Watch()
    go d.bugNotifier.Watch()

    return d.webServer.ListenAndServe()
}
```

### Configuration

```toml
# ~/.config/ntm/config.toml

[integrations]
# bv integration (NEW)
bv_track_assignment = true
bv_graph_insights = true
bv_auto_suggest = true

# CASS integration (NEW)
cass_history_injection = true
cass_history_limit = 5
cass_semantic_search = true

# s2p integration (NEW)
s2p_context_prep = true
s2p_token_budget = 100000
s2p_format = "xml"

# UBS integration (ENHANCED)
ubs_notifications = true
ubs_watch_interval = "30s"
ubs_dashboard_widget = true

# Existing integrations
caam_enabled = true
caam_auto_failover = true
cm_enabled = true
cm_auto_inject = true
slb_enabled = true
agent_mail_enabled = true

[daemon]
enabled = true
web_port = 8080
```

---

## Web Dashboard

### Enhanced Architecture with All Integrations

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           NTM Web Dashboard                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌────────────────────────────────────────────────────────────────┐     │
│  │                      Session Overview                           │     │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐               │     │
│  │  │ Agent 1 │ │ Agent 2 │ │ Agent 3 │ │ Agent 4 │               │     │
│  │  │ claude  │ │ claude  │ │ codex   │ │ gemini  │               │     │
│  │  │ Track A │ │ Track B │ │ Track C │ │ Track D │ ← bv Tracks   │     │
│  │  │ 🟢 alice│ │ 🟡 bob  │ │ 🟢 work │ │ 🟢 main │ ← CAAM        │     │
│  │  │ Working │ │ Idle    │ │ Working │ │ Waiting │               │     │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘               │     │
│  └────────────────────────────────────────────────────────────────┘     │
│                                                                         │
│  ┌──────────────────────┐  ┌──────────────────────┐  ┌──────────────┐  │
│  │   UBS Status (NEW)   │  │  Historical Context  │  │ Graph Metrics│  │
│  │   2 critical bugs    │  │  5 relevant sessions │  │ (bv insights)│  │
│  │   7 warnings         │  │  from CASS           │  │              │  │
│  │   [View All]         │  │  [Search More]       │  │ PageRank: #3 │  │
│  └──────────────────────┘  └──────────────────────┘  │ InDegree: 5  │  │
│                                                       └──────────────┘  │
│  ┌──────────────────────┐  ┌──────────────────────────────────────┐    │
│  │   SLB Pending (2)    │  │   Memory Context (CM)                │    │
│  │   ┌────────────────┐ │  │   247 rules in playbook              │    │
│  │   │ DANGEROUS      │ │  │   45 proven, 89 established          │    │
│  │   │ rm -rf ./build │ │  │                                      │    │
│  │   │[Approve][Reject]│ │  │   Active rules (this session):      │    │
│  │   └────────────────┘ │  │   - JWT validation first             │    │
│  └──────────────────────┘  │   - Use httptest for auth            │    │
│                            └──────────────────────────────────────┘    │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                    Agent Output Stream                            │   │
│  │  [Agent 1] Working on Track A (3 items remaining)                │   │
│  │  [Agent 1] // [cass: helpful b-8f3a2c] - JWT check helped        │   │
│  │  [UBS] ⚠️ Found 2 issues in auth.go (notified Agent 1)           │   │
│  │  [Agent 3] Tests passing: 47/50                                  │   │
│  │  [CASS] Injected 3 relevant historical snippets for Agent 2      │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Roadmap

### Phase 1: Underexplored Integrations (Highest ROI)

**bv Robot Modes** (Week 1)
- [ ] Track-based agent assignment
- [ ] Graph-based prioritization (PageRank, betweenness)
- [ ] Label-based work distribution
- [ ] Robot mode: `--robot-tracks`, `--robot-insights`

**CASS Historical Context** (Week 2)
- [ ] Pre-task context enrichment
- [ ] Project-specific history search
- [ ] Error resolution history
- [ ] Robot mode: `--robot-history`

**UBS Enhancements** (Week 3)
- [ ] Dashboard bug widget
- [ ] Agent bug notifications
- [ ] Session quality metrics
- [ ] Robot mode: `--robot-bugs`, `--robot-quality`

**s2p Context Preparation** (Week 4)
- [ ] Token-budgeted context prep
- [ ] Automatic file discovery
- [ ] Pipeline stage context
- [ ] Robot mode: `--robot-context`

### Phase 2: Planned Integrations

**CAAM Integration** (Month 2)
- [ ] Account assignment on spawn
- [ ] Rate limit detection and failover
- [ ] Health status in robot mode
- [ ] Account status in dashboard

**CM Integration** (Month 2)
- [ ] Pre-task context injection
- [ ] Post-session reflection trigger
- [ ] Memory status in robot mode
- [ ] Rule tracking in dashboard

**SLB Integration** (Month 2)
- [ ] SLB-wrapped command execution
- [ ] Approval dispatch to reviewers
- [ ] Pending requests in dashboard
- [ ] Emergency override via NTM

### Phase 3: Unified Experience

- [ ] NTM daemon for background coordination
- [ ] Unified configuration system
- [ ] `ntm go` zero-config command
- [ ] Multi-channel notifications

### Phase 4: Web Dashboard

- [ ] Real-time agent output streaming
- [ ] All integration widgets
- [ ] SLB approval UI
- [ ] Historical context browser

### Phase 5: IDE Integration

- [ ] VSCode extension
- [ ] Neovim plugin
- [ ] API documentation

---

## Success Metrics

### Integration Health

| Metric | Target | Measurement |
|--------|--------|-------------|
| bv track assignment usage | >80% of assignments | `--robot-assign` output |
| CASS context injection rate | >90% of spawns | Daemon logs |
| UBS notification delivery | >95% of findings | Agent Mail logs |
| s2p context budget compliance | <5% truncations | s2p output |
| CAAM failover success | >95% | Cooldown logs |
| CM rule hit rate | >60% | Agent feedback |
| SLB approval latency | <2 minutes | SLB logs |

### User Experience

| Metric | Target | Measurement |
|--------|--------|-------------|
| Time to first working session | <3 minutes | User testing |
| Zero-config success rate | >80% | Error logs |
| Dashboard adoption | >50% of users | Web server logs |

### Ecosystem Adoption

| Metric | Target | Measurement |
|--------|--------|-------------|
| All 8 stack tools integrated | 100% | Feature checklist |
| Cross-tool data flow | Verified | Integration tests |
| Cross-agent learning | Measurable | CM rule sources |

---

## Conclusion

NTM's position as the **cockpit** of the Agentic Coding Flywheel means it must integrate deeply with all ecosystem tools. This plan identifies **four major underexplored integration opportunities**:

1. **bv Robot Modes**: 12+ modes for graph analysis, track planning, and smart assignment
2. **CASS Historical Context**: Inject relevant past solutions before agents start work
3. **s2p Context Preparation**: Token-budgeted context to prevent overflow
4. **UBS Notifications**: Alert agents to bugs in files they're modifying

These integrations, combined with the **previously planned** CAAM, CM, and SLB integrations, will transform NTM from a session manager into an **intelligent orchestrator** that:

- Assigns work optimally (bv graph analysis)
- Provides historical context (CASS search)
- Prevents context overflow (s2p budgeting)
- Alerts to quality issues (UBS notifications)
- Manages rate limits transparently (CAAM)
- Learns from every session (CM)
- Gates dangerous operations (SLB)
- Coordinates all agents (Agent Mail)

The result is a closed-loop system where each cycle compounds, making the entire development flywheel spin faster and more reliably.

---

*Document generated: 2025-01-03*
*NTM Version: v1.3.0*
*Ecosystem: Dicklesworthstone Stack v1.0*
