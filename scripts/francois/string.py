list = [219,224,1145,"012",1130,1031,1042,172,175,177,"0824","022","026"]
value = [0,0,1,0,1,1,1,0,0,0,1,0,0]
ans = ""

for i in range (0,13):
    if value[i] == 0:
        ans = ans + " Francois@amd"+ str(list[i]) +".utah.cloudlab.us"
    else:
        ans = ans + " Francois@ms"+ str(list[i]) +".utah.cloudlab.us"
print(ans)