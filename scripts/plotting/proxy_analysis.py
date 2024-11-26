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

import pandas as pd
import matplotlib.pyplot as plt
import numpy as np

def parse_df(path):
    df = pd.read_csv(path)
    df = df[df['cold_start'] == 0]
    df = df.drop(['time', 'service_name', 'container_id'], axis=1)

    df['proxying'] = df['proxying'] - 1460

    return df

async_df = parse_df("proxy_async.csv").mean(axis=0)
sync_df = parse_df("proxy.csv").mean(axis=0)

labels = parse_df("proxy_async.csv").columns.values.tolist()
fig, (ax1,ax2) = plt.subplots(1,2)


ax1.title.set_text("Sync data plane | Total (microseconds) : " + str(int(np.array(sync_df).sum())))
ax1.pie(sync_df, labels=labels,autopct='%1.1f%%')

ax2.title.set_text("Async data plane | Total (microseconds) : " + str(int(np.array(async_df).sum())))
ax2.pie(async_df, labels=labels,autopct='%1.1f%%')


plt.show()