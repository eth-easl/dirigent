import argparse
parser=argparse.ArgumentParser(description="Custom address generator with arugments")
parser.add_argument('--type', required=False, default = "basic", type=str)
args=parser.parse_args()

start_addr = 0
if args.type == "worker":
    start_addr = 3
elif args.type == "worker-ha":
    start_addr = 7

# Change this line with the automatic script
list = ["Francois@pc841.emulab.net","Francois@pc790.emulab.net","Francois@pc738.emulab.net","Francois@pc734.emulab.net","Francois@pc848.emulab.net","Francois@pc737.emulab.net","Francois@pc795.emulab.net"]
ans = ""

for i in range (start_addr,len(list)):
    ans = ans + " " + str(list[i])

print(ans)