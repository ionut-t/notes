# Notes CLI

A simple, lightweight command-line notes manager written in Go.

## Features

- Create notes
- View and manage notes

- Customizable storage location and editor

## Installation

```bash
go install github.com/ionut-t/notes@latest
```

Or install the binary from the [Releases page](https://github.com/ionut-t/notes/releases).

## Usage

### Basic Commands

```bash
# Create a new note
notes add

# Launch the notes manager UI
notes

# Configure settings
notes config [flags]
```

### Configuration Options

```bash
# Open configuration file in your default editor
notes config

# Set custom editor
notes config --editor nvim

# Set custom storage location
notes config --storage ~/Documents/my-notes
```

## Directory Structure

```
~/.notes/              # Default storage location
├── .config.toml       # Configuration file
└── *.md               # Your markdown notes
```

## License

[MIT License](LICENSE)

