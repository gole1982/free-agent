# CI/CD Pipeline Documentation

## 1. Pipeline Overview

This document describes the Continuous Integration and Continuous Deployment (CI/CD) pipeline for the **Free Agent** project. The pipeline automates build, testing, and release processes across multiple platforms.

## 2. Trigger Events

| Event | Branch | Action |
|-------|--------|--------|
| `push` | `main`, `develop` | Full CI/CD pipeline |
| `pull_request` | `main`, `develop` | Build & Test only |

## 3. Pipeline Stages

### Stage 1: Build & Test

**Purpose:** Validate code quality and ensure cross-platform compatibility.

**Execution Matrix:**
- **Operating Systems:** Ubuntu 22.04, Windows Server 2022, macOS 13
- **Go Versions:** 1.22, 1.23

**Steps:**
1. **Checkout Code** - Retrieve latest commit from repository
2. **Set up Go** - Install specified Go version with cache optimization
3. **Install Dependencies** - `go mod tidy`
4. **Build** - Compile binary for target platform
5. **Run Tests** - Execute all unit tests
6. **Upload Artifacts** - Store build outputs for downstream stages

### Stage 2: Release (Main Branch Only)

**Purpose:** Automatically create GitHub releases for production builds.

**Steps:**
1. **Download Artifacts** - Retrieve build outputs from all platforms
2. **Create Release** - Publish binaries to GitHub Releases
3. **Tag Version** - Auto-increment version using run number

## 4. Artifact Management

| Platform | Artifact Name | File Path |
|----------|---------------|-----------|
| Linux | `free-agent-linux` | `bin/free-agent` |
| Windows | `free-agent-windows` | `bin/free-agent.exe` |
| macOS | `free-agent-macos` | `bin/free-agent` |

## 5. Environment Variables

| Variable | Source | Purpose |
|----------|--------|----------|
| `GITHUB_TOKEN` | GitHub Actions | Authentication for release creation |
| `GO_VERSION` | Matrix Configuration | Specify Go compiler version |

## 6. Security Considerations

- **Secrets Management:** All sensitive data stored in GitHub Secrets
- **Dependency Verification:** Uses Go module proxy for secure dependency resolution
- **No Credentials in Workflow:** No hardcoded credentials in pipeline configuration

## 7. Pipeline Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    GitHub Event Trigger                      │
│              (push / pull_request)                          │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│               Build & Test Stage                            │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌───────┐│
│  │Checkout │→│ Set Go  │→│ Deps    │→│ Build   │→│ Tests ││
│  │         │ │ Version │ │ Tidy    │ │ Binary  │ │       ││
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └───────┘│
│                         │                                  │
│                         ▼                                  │
│              ┌──────────────────────┐                      │
│              │ Upload Artifacts     │                      │
│              └──────────────────────┘                      │
└──────────────────────────┬──────────────────────────────────┘
                           │
                   ┌───────┴───────┐
                   │               │
          ┌────────▼────────┐ ┌────▼──────────┐
          │ PR Branch?      │ │ Main Branch?  │
          │ (Skip Release)  │ │ (Create       │
          └─────────────────┘ │  Release)     │
                              └──────┬────────┘
                                     │
                                     ▼
                    ┌───────────────────────────┐
                    │       Release Stage       │
                    │  ┌─────────────────────┐  │
                    │  │ Download Artifacts  │  │
                    │  │ Create Release      │  │
                    │  │ Tag Version         │  │
                    │  └─────────────────────┘  │
                    └───────────────────────────┘
```

## 8. Workflow Configuration Reference

### File Location
```
.github/workflows/ci-cd.yml
```

### Key Configuration Values

| Parameter | Value | Description |
|-----------|-------|-------------|
| `concurrency` | N/A | No concurrency limits |
| `timeout-minutes` | Default | GitHub Actions default timeout |
| `runs-on` | Matrix | Ubuntu/Windows/macOS |

## 9. Release Versioning

- **Tag Format:** `v{run_number}`
- **Example:** `v123`
- **Auto-increment:** Based on GitHub Actions run number
- **Release Notes:** Auto-generated with build artifacts

## 10. Failure Handling

- **Build Failure:** Pipeline stops immediately, notification via GitHub
- **Test Failure:** Pipeline stops, test results available in workflow logs
- **Release Failure:** Previous artifacts preserved, manual release possible

## 11. Manual Override

To trigger a manual release:

```bash
# Create tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push tag
git push origin v1.0.0
```

## 12. Pipeline Metrics

| Metric | Tracking Method |
|--------|-----------------|
| Build Time | GitHub Actions workflow duration |
| Test Coverage | Go test output |
| Success Rate | GitHub Actions dashboard |

---

*Document Version: 1.0*
*Last Updated: 2026-06-02*
