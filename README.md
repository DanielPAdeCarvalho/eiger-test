# eiger-test
---
# File Diff Utility

## Overview

This utility provides a powerful solution for detecting and applying changes between two files. It generates a delta representation of the changes needed to transform an original file into its updated version and applies these changes to produce a new, updated file. This tool is essential for scenarios involving version control, differential backups, or any application where tracking and applying file changes efficiently is crucial.

## Features

- **Delta Generation**: Efficiently computes the difference between the original and updated files, producing a compact delta representation of the changes.
- **Delta Application**: Applies the delta to the original file, creating an updated version that matches the target file.
- **Rolling Hash Algorithm**: Utilizes a rolling hash algorithm for optimized difference detection, facilitating quick and memory-efficient processing of large files.

## Installation

Clone the repository to your local machine:

```
git clone https://github.com/DanielPAdeCarvalho/eiger-test.git
```

Navigate to the project directory:

```
cd eiger-test
```

Build the project (ensure you have Go installed):

```
go build filediff-utility
```

## Usage

To use this utility, run the compiled executable from the command line, providing paths to the original file, the updated file, and the output file where the results will be saved:

```
./filediff-utility <originalFilePath> <updatedFilePath> <outputFilePath>
```

Ensure you replace `<originalFilePath>`, `<updatedFilePath>`, and `<outputFilePath>` with your specific file paths.

## How It Works

1. **Delta Generation**: The utility first reads the original and updated files, breaking them into blocks and computing their hashes. It then identifies which blocks have changed and prepares a list of operations required to transform the original file into the updated version.

2. **Applying Delta**: With the delta instructions ready, the utility reads the original file and applies the changes step by step, either copying unchanged blocks from the original file or inserting new data from the updated file, thereby creating the output file.

## Contributing

We welcome contributions to improve this project! Whether it's bug reports, feature suggestions, or direct code contributions, please feel free to make a pull request or open an issue on GitHub.

## License

This project is licensed under the MIT License - see the LICENSE.md file for details.

---
