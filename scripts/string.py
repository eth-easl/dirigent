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