# Documentation Structure

The reviewtask documentation is organized into two main sections:

## 📚 [User Guide](user-guide/README.md)
Documentation for developers using reviewtask to manage PR reviews.

**Contents:**
- Installation and setup
- Command reference
- Configuration options
- Workflow best practices
- Troubleshooting

## 🔧 [Developer Guide](developer-guide/README.md)
Documentation for contributors and developers extending reviewtask.

**Contents:**
- Architecture overview
- Development environment setup
- Project structure
- Testing guidelines
- Contributing process
- Release management

## 📂 Directory Structure

```
docs/
├── README.md                    # This file
├── index.md                     # Main documentation index
├── user-guide/                  # End-user documentation
│   ├── README.md               # User guide index
│   ├── installation.md         # Installation instructions
│   ├── quick-start.md          # Getting started guide
│   ├── authentication.md       # GitHub auth setup
│   ├── commands.md             # CLI command reference
│   ├── configuration.md        # Configuration guide
│   ├── workflow.md             # Best practices
│   └── troubleshooting.md      # Common issues
├── developer-guide/            # Developer documentation
│   ├── README.md              # Developer guide index
│   ├── architecture.md        # System architecture
│   ├── development-setup.md   # Dev environment setup
│   ├── project-structure.md   # Code organization
│   ├── testing.md             # Testing strategy
│   ├── contributing.md        # Contribution guidelines
│   ├── versioning.md          # Release process
│   ├── prd.md                 # Product requirements
│   └── implementation-progress.md # Feature tracking
└── assets/                     # Documentation assets
    └── images/                 # Screenshots and diagrams