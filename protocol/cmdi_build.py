import csv

csv_file = 'cmdid.csv'
go_file = 'CmdID.go'

code_map = {}
proto_map = {}

# 读取.csv文件
with open(csv_file, 'r') as file:
    reader = csv.reader(file)
    for row in reader:
        if len(row) == 2:
            code_map[row[1]] = row[0]

# 读取.csv文件
with open(csv_file, 'r') as file:
    reader = csv.reader(file)
    for row in reader:
        if len(row) == 2:
            proto_map[row[0]] = row[1]

# 生成CMDID Go
go_code = 'func (c *CmdProtoMap) registerMessage() {\n'
for key, value in code_map.items():
    go_code += '\tc.regMsg({}, func() any{{ return new(proto.{}) }})\n'.format(value, value)
go_code += '\t\n}\n'

# 保存到.go文件
with open(go_file, 'w') as file:
    file.write(go_code)

print("CMDID Go已经保存到{}".format(go_file))