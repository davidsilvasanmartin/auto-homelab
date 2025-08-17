from pathlib import Path
import subprocess
import shutil
import os
from typing import List, Optional, Union
from dotenv import load_dotenv


class CommandRunner:
    @staticmethod
    def run(command: str):
        try:
            process = subprocess.run(command, shell=True, check=True, capture_output=True, text=True)
            print(f"Successfully ran command: {command}")

            # Print the stdout if available
            if process.stdout:
                print("Command stdout output:")
                print(process.stdout)

            # If there's stderr output but the command didn't fail, it might be warnings
            if process.stderr:
                print("Command stderr output:")
                print(process.stderr)

            return process

        except subprocess.CalledProcessError as e:
            print(f"Error executing command: {e}")
            print(f"Stdout: {e.stdout}")
            print(f"Stderr: {e.stderr}")
            raise e


class Backup:
    """
    Base class for all backup operations.
    """
    def __init__(self, output_path: Path):
        """
        Initialize backup configuration.

        Args:
            output_path: Path where the backup should be stored
        """
        self.output_path = output_path

    def run(self) -> Union[Path, None]:
        """
        Execute the backup operation.

        Returns:
            Path to the backup or None
        """
        # Ensure backup directory exists
        self.output_path.mkdir(parents=True, exist_ok=True)
        print(f"Base backup operation to {self.output_path}")
        return None

    def _prepare_output_directory(self):
        """Helper method to prepare the output directory"""
        self.output_path.mkdir(parents=True, exist_ok=True)


class DirectoryBackup(Backup):
    def __init__(self, source_path: Path, output_path: Path, pre_command: Optional[str] = None, post_command: Optional[str]=None):
        """
        Initialize directory backup configuration. It copies the contents of source_path to target_dir.

        Args:
            source_path: The directory to backup
            output_path: The directory where the backup should be stored
            pre_command: The command to execute before copying the directory. This will typically generate the data we want to backup
            post_command: The command to execute after copying the directory.
        """
        super().__init__(output_path)
        self.source_path = source_path
        self.pre_command = pre_command
        self.post_command = post_command

    def run(self) -> Path:
        """
        Execute the directory backup operation.

        Returns:
            Path to the backup destination
        """
        # Call parent method to ensure directory exists
        super().run()

        # Run pre-command if provided
        if self.pre_command is not None:
            CommandRunner.run(self.pre_command)

        if self.source_path.exists() and self.source_path.is_dir():
            print(f"Copying '{self.source_path}' to '{self.output_path}'...")
            try:
                # Copy the entire directory tree
                shutil.copytree(self.source_path, self.output_path, dirs_exist_ok=True)
                print(f"Successfully copied '{self.source_path}' to '{self.output_path}'.\n")

                # Run post-command if provided
                if self.post_command is not None:
                    CommandRunner.run(self.post_command)

                return self.output_path
            except Exception as e:
                print(f"Error copying directory '{self.source_path}' to '{self.output_path}': {e}")
                raise e
        else:
            print(f"Source directory '{self.source_path}' does not exist or is not a directory. Skipping.")
            return self.output_path


class PostgreSQLBackup(Backup):
    """
    Class for backing up PostgreSQL databases using docker exec.
    """
    def __init__(
            self,
            container_name: str,
            db_name: str,
            username: str,
            password: str,
            output_path: Path
    ):
        """
        Initialize PostgreSQL backup configuration.

        Args:
            container_name: Name of the PostgreSQL container
            db_name: Name of the database to backup
            username: PostgreSQL username
            password: PostgreSQL password
            output_path: Path where the backup should be stored
        """
        super().__init__(output_path)
        self.container_name = container_name
        self.db_name = db_name
        self.username = username
        self.password = password

    def run(self) -> Path:
        """
        Execute the PostgreSQL backup using docker exec.

        Returns:
            Path to the created backup file
        """
        # Call parent method to ensure directory exists
        super().run()

        backup_file = self.output_path / f"{self.db_name}.sql"

        # Construct the docker command
        docker_command = (
            f"docker exec -i {self.container_name} /bin/bash -c "
            f"\"PGPASSWORD={self.password} pg_dump --username {self.username} {self.db_name}\""
            f" > {backup_file}"
        )

        print(f"Running backup command for database {self.db_name} in container {self.container_name}")
        try:
            CommandRunner.run(docker_command)
            print(f"Successfully backed up {self.db_name} to {backup_file}\n")
            return backup_file

        except subprocess.CalledProcessError as e:
            print(f"Error backing up database {self.db_name}: {e}")
            raise e


class MySQLBackup(Backup):
    """
    Class for backing up MySQL databases using docker exec.
    """
    def __init__(
            self,
            container_name: str,
            db_name: str,
            username: str,
            password: str,
            output_path: Path
    ):
        """
        Initialize MySQL backup configuration.

        Args:
            container_name: Name of the MySQL container
            db_name: Name of the database to backup
            username: MySQL username
            password: MySQL password
            output_path: Path where the backup should be stored
        """
        super().__init__(output_path)
        self.container_name = container_name
        self.db_name = db_name
        self.username = username
        self.password = password

    def run(self) -> Path:
        """
        Execute the MySQL backup using docker exec.

        Returns:
            Path to the created backup file
        """
        # Call parent method to ensure directory exists
        super().run()

        backup_file = self.output_path / f"{self.db_name}.sql"

        # Construct the docker command
        docker_command = (
            f"docker exec -i {self.container_name} /bin/bash -c "
            f"\"MYSQL_PWD={self.password} mysqldump --user {self.username} {self.db_name}\""
            f" > {backup_file}"
        )

        print(f"Running backup command for database {self.db_name} in container {self.container_name}")
        try:
            CommandRunner.run(docker_command)
            print(f"Successfully backed up {self.db_name} to {backup_file}\n")
            return backup_file

        except subprocess.CalledProcessError as e:
            print(f"Error backing up database {self.db_name}: {e}")
            raise e

class MariaDbBackup(Backup):
    """
    Class for backing up MariaDB databases using docker exec. It is very similar to the MySQLBackup class,
    but uses mariadb-dump instead of mysqldump.
    """
    def __init__(
            self,
            container_name: str,
            db_name: str,
            username: str,
            password: str,
            output_path: Path
    ):
        """
        Initialize MariaDB backup configuration.

        Args:
            container_name: Name of the MariaDB container
            db_name: Name of the database to backup
            username: MariaDB username
            password: MariaDB password
            output_path: Path where the backup should be stored
        """
        super().__init__(output_path)
        self.container_name = container_name
        self.db_name = db_name
        self.username = username
        self.password = password

    def run(self) -> Path:
        """
        Execute the MariaDB backup using docker exec and mariadb-dump.

        Returns:
            Path to the created backup file
        """
        # Call parent method to ensure directory exists
        super().run()

        backup_file = self.output_path / f"{self.db_name}.sql"

        # Construct the docker command using mariadb-dump
        docker_command = (
            f"docker exec -i {self.container_name} /bin/bash -c "
            f"\"MYSQL_PWD={self.password} mariadb-dump --user {self.username} {self.db_name}\""
            f" > {backup_file}"
        )

        print(f"Running backup command for MariaDB database {self.db_name} in container {self.container_name}")
        try:
            CommandRunner.run(docker_command)
            print(f"Successfully backed up {self.db_name} to {backup_file}\n")
            return backup_file

        except subprocess.CalledProcessError as e:
            print(f"Error backing up MariaDB database {self.db_name}: {e}")
            raise e



def prepare_backup_directory(output_path: Path):
    """Prepare an empty backup directory"""
    output_path.mkdir(parents=True, exist_ok=True)

    # Clear contents of the backup directory
    print(f"Preparing backup directory: {output_path}")
    for item in output_path.iterdir():
        try:
            if item.is_file() or item.is_symlink():
                item.unlink()
                print(f"Removed file/symlink: {item}")
            elif item.is_dir():
                shutil.rmtree(item)
                print(f"Removed sub-directory: {item}")
        except OSError as e:
            print(f"Error removing item {item} from {output_path}: {e}")
            raise e
    print(f"Successfully prepared directory {output_path}.\n")

def get_required_env_var(var: str) -> str:
    """Get required environment variable"""
    try:
        return os.environ[var]
    except KeyError as e:
        raise ValueError(f"Missing required environment variable: {e}")

if __name__ == "__main__":
    # Load environment variables from .env file
    load_dotenv()

    main_backup_dir = Path(get_required_env_var("HOMELAB_BACKUP_PATH"))
    # Prepare the main backup directory
    prepare_backup_directory(main_backup_dir)

    # Define the source directories and their target names using the new DirectoryBackup class
    # TODO need to somewhat check that necessary docker containers are running
    backup_operations: List[Backup] = [
        DirectoryBackup(
            source_path=Path(get_required_env_var("HOMELAB_CALIBRE_LIBRARY_PATH")),
            output_path=main_backup_dir / "calibre-web-automated-calibre-library"
        ),
        # NOTE: arguments of docker compose are SERVICE names, and NOT CONTAINER names
        DirectoryBackup(
            pre_command="docker compose stop calibre",
            source_path=Path(get_required_env_var("HOMELAB_CALIBRE_CONF_PATH")),
            output_path=main_backup_dir / "calibre-web-automated-config",
            post_command="docker compose start calibre",
        ),
        DirectoryBackup(
            pre_command="docker compose start paperless-redis paperless-db paperless && docker compose exec -T paperless document_exporter -d ../export",
            source_path=Path(get_required_env_var("HOMELAB_PAPERLESS_WEB_EXPORT_PATH")),
            output_path=main_backup_dir / "paperless-ngx-webserver-export",
        ),
        PostgreSQLBackup(
            container_name=get_required_env_var("HOMELAB_IMMICH_DB_CONTAINER_NAME"),
            db_name=get_required_env_var("HOMELAB_IMMICH_DB_DATABASE"),
            username=get_required_env_var("HOMELAB_IMMICH_DB_USER"),
            password=get_required_env_var("HOMELAB_IMMICH_DB_PASSWORD"),
            output_path=main_backup_dir / "immich-db"
        ),
        DirectoryBackup(
            source_path=Path(get_required_env_var("HOMELAB_IMMICH_WEB_UPLOAD_PATH")),
            output_path=main_backup_dir / "immich-library"
        ),
        MariaDbBackup(
            container_name=get_required_env_var("HOMELAB_FIREFLY_DB_CONTAINER_NAME"),
            db_name=get_required_env_var("HOMELAB_FIREFLY_DB_DATABASE"),
            username=get_required_env_var("HOMELAB_FIREFLY_DB_USER"),
            password=get_required_env_var("HOMELAB_FIREFLY_DB_PASSWORD"),
            output_path=main_backup_dir / "firefly-db"
        )
    ]

    # Run all backup operations
    for operation in backup_operations:
        operation.run()

    print("Backup process finished.")