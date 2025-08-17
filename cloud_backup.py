import os
import subprocess
import argparse
import datetime
import sys
from pathlib import Path
from dotenv import load_dotenv


class ResticToBackblazeBackup:
    """
    Class to handle backups to Backblaze B2 using restic
    """
    def __init__(self, backup_path:str, repository_url:str, b2_account_id:str, b2_account_key:str,
                 restic_password:str, retention_days:int):
        """
        Initialize the class with configuration parameters.

        Args:
            backup_path (str): Path to the directory to backup
            repository_url (str): Restic repository URL (B2 bucket path)
            b2_account_id (str): Backblaze B2 Account ID
            b2_account_key (str): Backblaze B2 Account Key
            restic_password (str): Password for the restic repository
            retention_days (int): Number of days to keep backups
        """
        self.backup_path = Path(backup_path).resolve()
        self.repository = repository_url
        self.b2_account_id = b2_account_id
        self.b2_account_key = b2_account_key
        self.restic_password = restic_password
        self.retention_days = retention_days

        # Validate required parameters
        self._validate_config()

        # Set environment variables for restic
        self.env = os.environ.copy()
        self.env['RESTIC_REPOSITORY'] = self.repository
        self.env['B2_ACCOUNT_ID'] = self.b2_account_id
        self.env['B2_ACCOUNT_KEY'] = self.b2_account_key
        self.env['RESTIC_PASSWORD'] = self.restic_password

    def _validate_config(self):
        """Validate that all required configuration parameters are present"""
        missing_params = []
        if not self.repository:
            missing_params.append('repository_url')
        if not self.b2_account_id:
            missing_params.append('b2_account_id')
        if not self.b2_account_key:
            missing_params.append('b2_account_key')
        if not self.restic_password:
            missing_params.append('restic_password')
        if not self.retention_days:
            missing_params.append('retention_days')

        if missing_params:
            raise ValueError(f"Missing required parameters: {', '.join(missing_params)}")

        if not self.backup_path.exists():
            raise ValueError(f"Backup path does not exist: {self.backup_path}")

    def _run_command(self, command, check=True):
        """Run a shell command with the configured environment variables"""
        try:
            result = subprocess.run(
                command,
                env=self.env,
                check=check,
                capture_output=True,
                text=True
            )
            return result
        except subprocess.CalledProcessError as e:
            print(f"Command failed: {' '.join(command)}")
            print(f"Error: {e}")
            print(f"Stdout: {e.stdout}")
            print(f"Stderr: {e.stderr}")
            if check:
                raise e
            return e

    def initialize_repository(self):
        """Initialize a new restic repository if it doesn't exist"""
        print("Checking if repository exists...")
        result = self._run_command(['restic', 'snapshots'], check=False)

        if result.returncode != 0 and "unable to open config file" in result.stderr:
            print("Repository does not exist. Initializing...")
            self._run_command(['restic', 'init'])
            print("Repository initialized successfully.")
        else:
            print("Repository already exists.")

    def create_backup(self):
        """Create a new backup of the specified directory"""
        print(f"Creating backup of {self.backup_path}...")

        # Add timestamp tag to the backup
        timestamp = datetime.datetime.now().strftime("%Y-%m-%d_%H-%M-%S")

        result = self._run_command([
            'restic', 'backup',
            str(self.backup_path),
            '--tag', f'automatic-{timestamp}',
            '--verbose'
        ])

        print("Backup completed successfully.")
        return result

    def prune_old_backups(self):
        """Remove old backups according to retention policy"""
        print(f"Pruning backups older than {self.retention_days} days...")

        # Keep daily backups for the specified retention period
        result = self._run_command([
            'restic', 'forget',
            '--keep-within', f"{self.retention_days}d",
            '--prune'
        ])

        print("Pruning completed successfully.")
        return result

    def check_repository(self):
        """Check repository for errors"""
        print("Checking repository integrity...")
        result = self._run_command(['restic', 'check'])
        print("Repository check completed.")
        return result

    def list_snapshots(self):
        """List all snapshots in the repository"""
        print("Listing snapshots...")
        result = self._run_command(['restic', 'snapshots'])
        print(result.stdout)
        return result

    def list_files_in_snapshot(self, snapshot_id):
        """
        List all files contained in a specific snapshot

        Args:
            snapshot_id (str): ID of the snapshot to list files from
        """
        print(f"Listing files in snapshot {snapshot_id}...")
        result = self._run_command(['restic', 'ls', snapshot_id])
        print(result.stdout)
        return result

    def restore_latest(self, target_dir):
        """
        Restore the latest snapshot to the specified directory

        Args:
            target_dir (str): Directory where files should be restored
        """
        target_path = Path(target_dir)
        # Ensure the target directory exists
        target_path.mkdir(parents=True, exist_ok=True)

        print(f"Restoring latest snapshot to {target_path}...")
        result = self._run_command([
            'restic', 'restore', 'latest',
            '--target', str(target_path),
            '--verbose'
        ])

        if result.returncode == 0:
            print(f"Successfully restored latest snapshot to {target_path}")
        else:
            print(f"Failed to restore snapshot to {target_path}")

        return result

    def run_full_backup(self):
        """Run a complete backup workflow"""
        try:
            self.initialize_repository()
            self.create_backup()
            self.prune_old_backups()
            return True
        except Exception as e:
            print(f"Backup failed: {e}")
            return False


def get_config_from_env()->dict:
    """Get backup configuration from environment variables"""

    load_dotenv()
    try :
        return {
            'repository_url': os.environ['HOMELAB_BACKUP_RESTIC_REPOSITORY'],
            'b2_account_id': os.environ['HOMELAB_BACKUP_B2_ACCOUNT_ID'],
            'b2_account_key': os.environ['HOMELAB_BACKUP_B2_ACCOUNT_KEY'],
            'restic_password': os.environ['HOMELAB_BACKUP_RESTIC_PASSWORD'],
            'backup_path': os.environ['HOMELAB_BACKUP_PATH'],
            'retention_days': int(os.environ['HOMELAB_BACKUP_RETENTION_DAYS']),
        }
    except KeyError as e:
        raise ValueError(f"Missing required environment variable: {e}")


def main():
    parser = argparse.ArgumentParser(description='Backup to Backblaze B2 using restic')
    parser.add_argument('--command', required=True,
                        choices=['backup', 'init', 'check', 'list', 'prune', 'restore', 'ls-files'],
                        help='Command to run')
    parser.add_argument('--restore-dir',
                        help='Directory where to restore files (only used with restore command)')
    parser.add_argument('--snapshot-id',
                        help='Snapshot ID to list files from (required for ls-files command)')

    args = parser.parse_args()

    try:
        # Get configuration from environment variables
        config = get_config_from_env()

        # Create backup instance with all parameters explicitly specified
        backup = ResticToBackblazeBackup(
            backup_path=config['backup_path'],
            repository_url=config['repository_url'],
            b2_account_id=config['b2_account_id'],
            b2_account_key=config['b2_account_key'],
            restic_password=config['restic_password'],
            retention_days=config['retention_days']
        )

        if args.command == 'backup':
            success = backup.run_full_backup()
            sys.exit(0 if success else 1)
        elif args.command == 'init':
            backup.initialize_repository()
        elif args.command == 'check':
            backup.check_repository()
        elif args.command == 'list':
            backup.list_snapshots()
        elif args.command == 'prune':
            backup.prune_old_backups()
        elif args.command == 'restore':
            if not args.restore_dir:
                print("Error: --restore-dir is required for restore command")
                sys.exit(1)
            backup.restore_latest(args.restore_dir)
        elif args.command == 'ls-files':
            if not args.snapshot_id:
                print("Error: --snapshot-id is required for ls-files command")
                sys.exit(1)
            backup.list_files_in_snapshot(args.snapshot_id)

    except ValueError as e:
        print(f"Configuration error: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"An unexpected error occurred: {e}")
        sys.exit(1)

if __name__ == '__main__':
    main()