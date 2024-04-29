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
list = ["Francois@hp122.utah.cloudlab.us","Francois@hp123.utah.cloudlab.us","Francois@hp133.utah.cloudlab.us","Francois@hp130.utah.cloudlab.us","Francois@hp138.utah.cloudlab.us","Francois@hp143.utah.cloudlab.us","Francois@hp155.utah.cloudlab.us","Francois@hp121.utah.cloudlab.us","Francois@hp124.utah.cloudlab.us","Francois@hp159.utah.cloudlab.us"]
ans = ""

for i in range (start_addr,len(list)):
    ans = ans + " " + str(list[i])

print(ans)