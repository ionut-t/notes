# Notes CLI

A simple notes manager written in Go.

## Installation

```bash
go install github.com/ionut-t/notes@latest
```

## Usage

### Basic Commands

```bash
# Create a new note
notes add

# Launch the notes app (shows list of notes)
notes # or notes list
```

## Configuration

Notes are stored in `~/.notes` directory by default. Each note is saved as a separate Markdown file.

The application uses the editor specified in your `EDITOR` environment variable. If not set, it defaults to `vim`.

## License

[MIT License](LICENSE)

