set -x

python_bin=/home/jvirtane/.pyenv/versions/3.11.4/bin/python3.11

without_locking_dir=$1
with_locking_dir=$2
output_dir=$3

#without_locking_dir=process_baseline_results_1_10m_false_2500
#with_locking_dir=process_baseline_results_1_10m_true_2500

without_locking_baseline_time="$(cat "${without_locking_dir}/baseline_time.txt")"
with_locking_baseline_time="$(cat "${with_locking_dir}/baseline_time.txt")"

for container in central central-db sensor; do
  $python_bin plot.py ${without_locking_dir}/metrics_${container}_mem.txt "Without locking" ${with_locking_dir}/metrics_${container}_mem.txt "With locking" "${container} memory" "Memory usage" "$without_locking_baseline_time" "$with_locking_baseline_time" ${output_dir}/${container}_mem_usage.png
  $python_bin plot.py ${without_locking_dir}/metrics_${container}_cpu.txt "Without locking" ${with_locking_dir}/metrics_${container}_cpu.txt "With locking" "${container} CPU" "CPU usage" "$without_locking_baseline_time" "$with_locking_baseline_time" ${output_dir}/${container}_cpu_usage.png
done

for table in process_indicators process_baselines process_baseline_results alerts deployments; do
  $python_bin plot.py ${without_locking_dir}/metrics_${table}.txt "Without locking" ${with_locking_dir}/metrics_${table}.txt "With locking" "${table} size" "${table} size" "$without_locking_baseline_time" "$with_locking_baseline_time" ${output_dir}/${table}_size.png
  $python_bin plot.py ${without_locking_dir}/metrics_${table}_bytes.txt "Without locking" ${with_locking_dir}/metrics_${table}_bytes.txt "With locking" "${table} size bytes" "${table} size bytes" "$without_locking_baseline_time" "$with_locking_baseline_time" ${output_dir}/${table}_size_bytes.png
done
