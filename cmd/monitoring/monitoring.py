#  MIT License
#
#  Copyright (c) 2024 EASL
#
#  Permission is hereby granted, free of charge, to any person obtaining a copy
#  of this software and associated documentation files (the "Software"), to deal
#  in the Software without restriction, including without limitation the rights
#  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
#  copies of the Software, and to permit persons to whom the Software is
#  furnished to do so, subject to the following conditions:
#
#  The above copyright notice and this permission notice shall be included in all
#  copies or substantial portions of the Software.
#
#  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
#  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
#  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
#  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
#  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
#  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
#  SOFTWARE.

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
