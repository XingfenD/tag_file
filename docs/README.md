# tagfile

A cross-platform file organization and retrieval CLI tool. Tag files in any directory with configurable categories and search them instantly.

## Installation

```bash
git clone https://github.com/XingfenD/tag_file.git
cd tag_file
make install
```

Or build directly:

```bash
go build -o tagfile.out ./cmd/cli
```

## Quick Start

Initialize a workspace with demo data:

```bash
make setup
```

Then enter the workspace:

```bash
cd temp/test_workspace
```

## Usage

### 1. Create Categories

Categories group related tags, e.g. "Animal Type", "Scene Content".

```bash
tagfile category add "Animal Type"
tagfile category add "Scene Content"
tagfile category add "File Type"

tagfile category list
```

### 2. Create Tags

Tags belong to a category. Tag names must be unique within the same category.

```bash
tagfile tag add "Animal Type" Cat
tagfile tag add "Animal Type" Dog
tagfile tag add "Scene Content" Playing
tagfile tag add "Scene Content" Outdoor
tagfile tag add "File Type" Video
tagfile tag add "File Type" Image

tagfile tag list                    # list all tags
tagfile tag list "Animal Type"      # filter by category
```

### 3. Scan Files

Register files from a directory into the database.

```bash
tagfile file scan .            # scan current directory
tagfile file scan /path/to/dir # scan a specific directory

tagfile file list              # view registered files
```

### 4. Tag Files

```bash
tagfile file tag video1.mp4 Cat
tagfile file tag video1.mp4 Playing
tagfile file tag video1.mp4 Video

tagfile file info video1.mp4        # view file details and tags
tagfile file untag video1.mp4 Playing  # remove a tag
```

### 5. Search Files

Search files by tags with AND/OR/NOT logic.

```bash
# OR search (default): files matching any tag
tagfile search tag Cat Dog

# AND search: files must have all tags
tagfile search tag --and Cat Playing

# NOT search: exclude files with certain tags
tagfile search tag --not Scenery

# Combined: include Dog but exclude Golden Retriever
tagfile search tag Dog --not "Golden Retriever"
```

### 6. Import External Files

Import files from other locations into the current directory.

```bash
tagfile file import /path/to/external/video.mp4
```

### 7. Attribute Management (Advanced)

Define fine-grained attributes for tags.

```bash
# Add an attribute to a tag
tagfile attribute add Cat Breed

# Add allowed values for the attribute
tagfile attribute value add 1 "British Shorthair"
tagfile attribute value add 1 Ragdoll

# View attributes and values
tagfile attribute list Cat
tagfile attribute value list 1
```

### 8. Configuration

```bash
tagfile config list
tagfile config set mykey myvalue
tagfile config get mykey
tagfile config unset mykey
```

## Command Overview

```
tagfile help

  category   Manage categories
    add      Add a category
    remove   Remove a category
    list     List all categories

  tag        Manage tags
    add      Add a tag
    remove   Remove a tag
    list     List tags

  file       Manage files
    scan     Scan directory and register files
    import   Import external files
    list     List registered files
    tag      Tag a file
    untag    Remove a tag from a file
    info     View file details
    search   Search files by name

  search     Search files by tags
    tag      Tag-based search (--and / --or / --not)

  attribute  Manage tag attributes
    add      Add an attribute definition
    remove   Remove an attribute
    list     List attributes
    value    Manage attribute values
      add    Add an attribute value
      remove Remove an attribute value
      list   List attribute values

  config     Manage configuration
    set      Set a config value
    get      Get a config value
    list     List all config entries
    unset    Remove a config entry
```

## How It Works

Each directory has its own `.tag_file.db` (SQLite). Files are indexed in-place (no moving). External imports are copied into the current directory.

```bash
# In the directory you want to manage
cd ~/my_videos
tagfile category add "Animal Type"
tagfile tag add "Animal Type" Cat
tagfile file scan .
tagfile file tag video.mp4 Cat
tagfile search tag Cat
```

## Development

```bash
make build    # build
make test     # run tests (47 total)
make setup    # create demo workspace
make clean    # clean up
make install  # install to GOPATH
```

Standard Go tests are also available:

```bash
go test ./internal/service/... -v
```
