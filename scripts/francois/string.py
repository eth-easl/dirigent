list = [219,224,1145,"012",1130,1031,1042,172,175,177,"0824","022","026",173,174,234,"023","025",237,238,211,"0804",1122,"0829","0838","0826","009","021",1016,"019"]
value = [0,0,1,0,1,1,1,0,0,0,1,0,0,0,0,0,0,0,0,0,0,1,1,1,1,1,0,0,1,0]
ans = ""

for i in range (3,25):
    if value[i] == 0:
        ans = ans + " Francois@amd"+ str(list[i]) +".utah.cloudlab.us"
    else:
        ans = ans + " Francois@ms"+ str(list[i]) +".utah.cloudlab.us"
print(ans)