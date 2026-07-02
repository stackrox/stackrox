import pandas as pd
import awswrangler as wr
#import plotly.express as px

class PerfTest():
    def __init__(self, num_ns, num_deployments, num_pods, uuid):
        self.num_ns = num_ns
        self.num_deployments = num_deployments
        self.num_pods = num_pods
        self.uuid = uuid

    def str(self):
        return "Namespaces= " + str(self.num_ns) + " Deployments= " + str(self.num_deployments) + " Pods= " + str(self.num_pods) + " Total Pods= " + str(self.num_ns * self.num_deployments * self.num_pods)

aws_os_client = wr.opensearch.connect(
    host='',
    username='',
    password='',
)

#pd.options.plotting.backend = 'plotly'

def get_metric_for_test(test_uuid, metric_name, limit=500):
  df = wr.opensearch.search_by_sql(
        aws_os_client,
        sql_query=f"""
          SELECT
            *
          FROM
            kube-burner
          WHERE
            uuid = '{test_uuid}'
            AND metricName = '{metric_name}'
          ORDER BY timestamp DESC
          LIMIT {limit}
        """
      )

  return df


perf_test1 = PerfTest(10, 5, 1, '')
perf_test2 = PerfTest(100, 5, 1, '')
perf_test3 = PerfTest(200, 5, 1, '')
perf_test4 = PerfTest(500, 5, 1, '')

perf_tests = [perf_test1, perf_test2, perf_test3, perf_test4]


for perf_test in perf_tests:
    df = get_metric_for_test(perf_test.uuid, 'stackrox_container_memory')
    ave = df['value'].mean()
    
    print(perf_test.str(), "ave=", ave)
