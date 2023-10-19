#include <bits/stdc++.h>
#define int long long
using namespace std;

signed main() {
    ios_base::sync_with_stdio(false);
    cin.tie(0);

    std::ifstream infile("notes.txt");

    map<string, int> mp;

    string s;
    while(infile >> s) {
        mp[s]++;
    }

    int maxValue = 0;
    for(auto p : mp) {
        maxValue = max(maxValue, p.second);
    }

    cout << "Max TP is : " << maxValue << endl;
}