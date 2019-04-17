#!/usr/bin/env bash

gcloud auth application-default login \
    --scopes=https://www.googleapis.com/auth/userinfo.email,https://www.googleapis.com/auth/userinfo.profile,https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/cloudkms
