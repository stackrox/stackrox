package mocks

//go:generate mockgen-wrapper SecurityHubAPI github.com/aws/aws-sdk-go/service/securityhub/securityhubiface

//go:generate mockgen-wrapper Tx,Row,Rows,BatchResults github.com/jackc/pgx/v5
