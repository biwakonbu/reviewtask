# Contributing to reviewtask

Thank you for your interest in contributing to reviewtask! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by our code of conduct:
- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive criticism
- Respect differing viewpoints and experiences

## How to Contribute

### Reporting Issues

1. Check if the issue already exists
2. Create a new issue with a clear title and description
3. Include steps to reproduce the problem
4. Add relevant labels to your issue

### Submitting Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature-name`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`go test ./...`)
6. Commit your changes with clear messages
7. Push to your fork
8. Create a Pull Request

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Run `gofmt` before committing
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions small and focused

### Testing

- Write tests for new features
- Maintain test coverage above 80%
- Run tests before submitting PR:
  ```bash
  go test ./...
  go test -race ./...
  ```

### Commit Messages

Follow conventional commit format:
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Test additions or modifications
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

Example:
```
feat: Add support for GitLab integration

- Implement GitLab API client
- Add configuration options
- Update documentation
```

## Release Process

### Version Bumping with PR Labels

When creating a PR, add one of these labels to indicate the type of version bump:

- **`release:major`** - Breaking changes (increment X.0.0)
  - Use when: Removing features, changing APIs, incompatible changes
  - Example: Removing CLI commands, changing config format

- **`release:minor`** - New features (increment x.Y.0)
  - Use when: Adding new functionality, backwards-compatible changes
  - Example: New commands, new options, performance improvements

- **`release:patch`** - Bug fixes (increment x.y.Z)
  - Use when: Fixing bugs, documentation updates, small improvements
  - Example: Fixing errors, typos, small optimizations

### Label Guidelines

1. **One Label Per PR**: Only add one release label per PR
2. **Add Early**: Add the label when creating the PR or during review
3. **Automatic Release**: When merged, the PR will trigger an automatic release
4. **Label Removal**: The label is automatically removed after release

### Examples

```markdown
# PR Title: Fix error handling in PR detection
Labels: release:patch, bug

# PR Title: Add support for multiple repositories
Labels: release:minor, enhancement

# PR Title: Redesign configuration system
Labels: release:major, breaking-change
```

## Pull Request Process

1. **Create PR with clear description**
   - Explain what changes you made
   - Reference any related issues
   - Add appropriate labels

2. **Add Release Label**
   - Choose one: `release:major`, `release:minor`, or `release:patch`
   - Based on the type of changes

3. **Wait for Review**
   - Address reviewer feedback
   - Keep PR up to date with main branch

4. **Automatic Release**
   - Once merged, release is created automatically
   - Version bump based on your label
   - Release notes generated from commits

## Getting Help

- Check the [documentation](docs/)
- Look at existing issues and PRs
- Ask questions in issues with the `question` label
- Review the [versioning guide](versioning.md)

## License

By contributing, you agree that your contributions will be licensed under the project's MIT License.