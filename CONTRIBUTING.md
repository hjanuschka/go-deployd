# Contributing to Go-Deployd

Thank you for your interest in contributing to Go-Deployd! This document provides guidelines and information for contributors.

## Ways to Contribute

- 🐛 **Report bugs** - Help us identify and fix issues
- 💡 **Suggest features** - Share ideas for improvements
- 📝 **Improve documentation** - Help make our docs better
- 🛠️ **Submit code** - Fix bugs or implement new features
- 🧪 **Write tests** - Improve our test coverage
- 🎨 **Improve UI/UX** - Enhance the admin dashboard

## Getting Started

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/hjanuschka/go-deployd.git
   cd go-deployd
   ```

2. **Install dependencies**
   ```bash
   # Go dependencies (automatically managed)
   go mod download
   
   # Dashboard dependencies
   cd dashboard && npm install && cd ..
   
   # JavaScript sandbox dependencies
   cd js-sandbox && npm install && cd ..
   ```

3. **Run in development mode**
   ```bash
   make dev
   ```

### Development Environment

- **Go**: 1.23 or later
- **Node.js**: 18 or later (for dashboard and JavaScript events)
- **Make**: For build automation

## Code Style

### Go Code
- Follow standard Go formatting (`gofmt`)
- Use descriptive variable and function names
- Include comments for exported functions
- Write tests for new functionality

### JavaScript Code
- Use ES6+ features
- Follow consistent indentation (2 spaces)
- Use meaningful variable names
- Add JSDoc comments where helpful

### Commit Messages
Use conventional commit format:
```
feat: add column-based storage support
fix: resolve WebSocket connection issue
docs: update API documentation
test: add unit tests for events system
```

## Pull Request Process

### Before Submitting

1. **Run tests**
   ```bash
   make test
   ```

2. **Check formatting**
   ```bash
   make fmt
   ```

3. **Build successfully**
   ```bash
   make build
   ```

4. **Update documentation** if needed

### Submitting a Pull Request

1. **Fork the repository** and create a feature branch
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** with clear, focused commits

3. **Test thoroughly** - include unit tests if applicable

4. **Update documentation** if your changes affect the API or usage

5. **Create a pull request** with:
   - Clear title and description
   - Reference to related issues
   - Screenshots (if UI changes)
   - Testing instructions

### Pull Request Requirements

- [ ] Code follows project style guidelines
- [ ] Tests pass (automated CI)
- [ ] Documentation updated (if applicable)
- [ ] No breaking changes (or clearly documented)
- [ ] Commit messages follow conventional format

## Development Guidelines

### Adding New Features

1. **Create an issue first** to discuss the feature
2. **Keep changes focused** - one feature per PR
3. **Add tests** for new functionality
4. **Update documentation** as needed
5. **Consider backward compatibility**

### Bug Fixes

1. **Reproduce the bug** with a test case
2. **Fix the issue** with minimal changes
3. **Verify the fix** doesn't break existing functionality
4. **Add regression tests** if possible

### Testing

#### Running Tests
```bash
# All tests
make test

# Specific package
go test ./internal/database/...

# With coverage
make test-coverage
```

#### Writing Tests
- Use table-driven tests for multiple test cases
- Mock external dependencies
- Test both success and error cases
- Include integration tests for complex features

### Database Changes

When modifying database functionality:
- Test with SQLite, MySQL, and MongoDB
- Ensure backward compatibility
- Update migration logic if needed
- Document any schema changes

### Event System Changes

When modifying the event system:
- Test both JavaScript and Go events
- Ensure hot-reload works correctly
- Update event documentation
- Test with different collection configurations

## Documentation

### API Documentation
- Update OpenAPI specs for API changes
- Include examples in documentation
- Test all documented examples

### Code Documentation
- Add comments for complex logic
- Document public APIs thoroughly
- Include usage examples where helpful

## Community Guidelines

### Code of Conduct
- Be respectful and inclusive
- Welcome newcomers and help them learn
- Focus on constructive feedback
- Assume good intentions

### Communication
- Use GitHub issues for bug reports and feature requests
- Join discussions in existing issues before creating new ones
- Be clear and specific in issue descriptions
- Include minimal reproduction cases for bugs

## Release Process

Releases are managed by maintainers and follow semantic versioning:
- **Major** (1.0.0): Breaking changes
- **Minor** (1.1.0): New features, backward compatible
- **Patch** (1.1.1): Bug fixes, backward compatible

## Getting Help

- 📖 **Documentation**: [docs/](./docs/)
- 🐛 **Issues**: [GitHub Issues](https://github.com/hjanuschka/go-deployd/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/hjanuschka/go-deployd/discussions)

## Recognition

Contributors are recognized in:
- Repository contributor lists
- Release notes for significant contributions
- Special thanks for major features or fixes

Thank you for contributing to Go-Deployd! 🚀