import psutil
import time

# Function to write utilization data to a CSV file
def write_utilization_to_csv():
    with open("utilization.csv", "w") as file:
        # Write the CSV header
        file.write("Timestamp,CPUUtilization,MEMUtilization,NETDownload,NETUpload\n")

    prev_bytes_sent, prev_bytes_recv = 0, 0
    while True:
        # Get the current Unix timestamp
        timestamp = int(time.time())

        # Get the CPU utilization
        cpu_percent = psutil.cpu_percent(interval=1)

        # Get the memory utilization
        memory_percent = psutil.virtual_memory().percent

        # Get network utilization
        net_io = psutil.net_io_counters()
        up = 0 if prev_bytes_sent == 0 else net_io.bytes_sent - prev_bytes_sent
        down = 0 if prev_bytes_recv == 0 else net_io.bytes_recv - prev_bytes_recv
        prev_bytes_sent, prev_bytes_recv = net_io.bytes_sent, net_io.bytes_recv

        # Append the utilization data to the CSV file
        with open("utilization.csv", "a") as file:
            file.write(f"{timestamp},{cpu_percent},{memory_percent},{down},{up}\n")

        time.sleep(1)  # Sleep for 1 second

if __name__ == "__main__":
    write_utilization_to_csv()
