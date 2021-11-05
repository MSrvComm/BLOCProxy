#!/usr/bin/env python3
out = ""
with open("proxy.sh", 'r') as proxy:
    lines = [line.strip() for line in proxy.readlines(
    ) if not line.startswith('#') and line.strip()]

    lines2 = [line.strip().split()
              for line in lines if not line.startswith('#') and line.strip()]

    for line in lines2:
        print(line)

    for line in lines:
        l = line.split()
        for i in range(len(l)):
            if i == 0:
                out += f"command:\n- {l[i]}\nargs:\n"
            else:
                out += f"- {l[i]}\n"
# print(out)
