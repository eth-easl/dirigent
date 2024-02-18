import argparse
parser=argparse.ArgumentParser(description="Custom address generator with arugments")
parser.add_argument('--type', required=False, default = "basic", type=str)
args=parser.parse_args()

start_addr = 0
if args.type != "basic":
    start_addr = 3

list = ["790","768","783","765","704", "788", "766", "796", "716", "772", "705", "792", "782"]
ans = ""

for i in range (start_addr,len(list)):
    #ans = ans + " Francois@hp"+ str(list[i]) +".utah.cloudlab.us"
    ans = ans + " Francois@pc"+ str(list[i]) +".emulab.net"

print(ans)