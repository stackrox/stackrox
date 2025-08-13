set -x

without_locking_dir=$1
with_locking_dir=$2
output_dir=$3

for num_deployments in 250 500 1000 2500 5000 10000 25000; do
	without_locking_dir=process_baseline_results/process_baseline_results_1_10m_${num_deployments}/process_baseline_results_1_10m_false_${num_deployments}
	with_locking_dir=process_baseline_results/process_baseline_results_1_10m_${num_deployments}/process_baseline_results_1_10m_true_${num_deployments}
	output_dir=process_baseline_results/process_baseline_results_1_10m_${num_deployments}
	./MakePlots.sh $without_locking_dir $with_locking_dir $output_dir
done
