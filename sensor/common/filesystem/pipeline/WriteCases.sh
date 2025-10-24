#!/usr/bin/env bash
set -eou pipefail

for operation in Creation Unlink Rename Permission Ownership Write Open; do
	caps_operation=${operation^^}
        echo -e "\tcase *sensorAPI.FileActivity_${operation}:"
        echo -e "\t\tactivity.File = &storage.FileActivity_File{"
        echo -e "\t\t\tPath:     fs.Get${operation}().GetActivity().GetPath(),"
        echo -e "\t\t\tHostPath: fs.Get${operation}().GetActivity().GetHostPath(),"
        echo -e "\t\t}"
        echo -e "\tactivity.Operation = storage.FileActivity_${caps_operation}"
done
