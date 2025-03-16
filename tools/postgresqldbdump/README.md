## Central DB Backup Dump Tool
This utility extracts data from the StackRox Central backups or its PostgreSQL dump.

### Installation
Ensure that you have [Go](https://golang.org/doc/install) installed on your system.
   ```bash
   go install ./tools/postgresqldbdump
   ```

### Usage
The tool supports two primary modes of operation:
1. **From a PostgreSQL Dump File:**
   ```bash
   go run . <path_to_postgres.dump> -d --output-dir <output_directory>
   ```
   Replace `<path_to_postgres.dump>` with the path to your PostgreSQL dump file and `<output_directory>` with your desired output directory.
2. **From a Central Backup File:**
   ```bash
   go run . <path_to_central_backup> --output-dir <output_directory>
   ```
   Replace `<path_to_central_backup>` with the path to your Central backup file and `<output_directory>` with your desired output directory.

### Options
- `-d`: Process the file as a PostgreSQL db dump.
- `--output-dir`: Specifies the directory where the extracted data will be saved.

### Example
```bash
go run . /path/to/central_backup.zip -d --output-dir /path/to/output
```
This command processes the specified Central DB PostgreSQL DB dump and saves the extracted data to the specified output directory.

