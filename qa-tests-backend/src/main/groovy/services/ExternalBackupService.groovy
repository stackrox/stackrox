package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.ExternalBackupServiceGrpc
import io.stackrox.proto.storage.ExternalBackupOuterClass.ExternalBackup
import io.stackrox.proto.storage.ExternalBackupOuterClass.S3Config
import io.stackrox.proto.storage.ExternalBackupOuterClass.GCSConfig
import io.stackrox.proto.storage.ScheduleOuterClass.Schedule
import util.Env

@CompileStatic
@Slf4j
class ExternalBackupService extends BaseService {
    static getExternalBackupClient() {
        return ExternalBackupServiceGrpc.newBlockingStub(getChannel())
    }

    static ExternalBackup getS3IntegrationConfig(
            String name,
            String bucket = Env.mustGetAWSS3BucketName(),
            String region = Env.mustGetAWSS3BucketRegion(),
            String endpoint = "",
            String accessKeyId = Env.mustGetAWSAccessKeyID(),
            String accessKey = Env.mustGetAWSSecretAccessKey())  {
        S3Config s3Config = S3Config.newBuilder()
                .setObjectPrefix(UUID.randomUUID().toString())
                .setBucket(bucket)
                .setRegion(region)
                .setEndpoint(endpoint)
                .setUseIam(false)
                .setAccessKeyId(accessKeyId)
                .setSecretAccessKey(accessKey)
                .build()

        return ExternalBackup.newBuilder()
                .setName(name)
                .setType("s3")
                .setBackupsToKeep(1)
                .setSchedule(Schedule.newBuilder()
                        .setIntervalType(Schedule.IntervalType.DAILY)
                        .setHour(0) //12:00 AM
                        .build()
                )
                .setS3(s3Config)
                .build()
    }

    static ExternalBackup getGCSIntegrationConfig(
            String name,
            Boolean useWorkloadId = false,
            String bucket = Env.mustGetGCSBucketName(),
            String serviceAccount = Env.mustGetGCSServiceAccount()) {

        GCSConfig gcsConfig = GCSConfig.newBuilder()
                .setObjectPrefix(UUID.randomUUID().toString())
                .setBucket(bucket)
                .setServiceAccount(useWorkloadId ? "" : serviceAccount)
                .setUseWorkloadId(useWorkloadId)
                .build()

        return ExternalBackup.newBuilder()
                .setName(name)
                .setType("gcs")
                .setBackupsToKeep(1)
                .setSchedule(Schedule.newBuilder()
                        .setIntervalType(Schedule.IntervalType.DAILY)
                        .setHour(0) //12:00 AM
                        .build()
                )
                .setGcs(gcsConfig)
                .build()
    }
}
