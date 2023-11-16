import psutil
import time

# Function to write utilization data to a CSV file
def write_utilization_to_csv():
    with open("utilization.csv", "w") as file:
        # Write the CSV header
        file.write("Timestamp, CPUUtilization, memoryUtilization \n")

    while True:
        # Get the current Unix timestamp
        timestamp = int(time.time())

        # Get the CPU utilization
        cpu_percent = psutil.cpu_percent(interval=1)

        # Get the memory utilization
        memory_percent = psutil.virtual_memory().percent

        # Append the utilization data to the CSV file
        with open("utilization.csv", "a") as file:
            file.write(f"{timestamp}, {cpu_percent}, {memory_percent}\n")

        time.sleep(1)  # Sleep for 1 second

if __name__ == "__main__":
    write_utilization_to_csv()
