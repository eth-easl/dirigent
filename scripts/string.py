import argparse
parser=argparse.ArgumentParser(description="Custom address generator with arugments")
parser.add_argument('--type', required=False, default = "basic", type=str)
args=parser.parse_args()

start_addr = 0
if args.type != "basic":
    start_addr = 3

# Change this line with the automatic script
list = ["Francois@pc852.emulab.net","Francois@pc856.emulab.net","Francois@pc845.emulab.net"]
ans = ""

for i in range (start_addr,len(list)):
    ans = ans + " " + str(list[i])

print(ans)