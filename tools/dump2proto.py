import os
import re

type_map = {
    'uint': 'uint32',
    'string': 'string',
    'ulong': 'uint64',
    'float': 'float',
    'int': 'int32',
    'double': 'double',
    'bool': 'bool',
    'long': 'int64',
    'HideType': 'uint32'  # 枚举类型映射
}

dump_cs_path = 'dump.cs'
dump_cs_lines = [line.strip() for line in open(dump_cs_path, 'r', encoding='utf-8').readlines()]
print(f'Open {dump_cs_path} with {len(dump_cs_lines)} lines')

# private readonly MapField<uint, AbyssSeasonInfo> seasonInfo_; // 0x18
re_dict_type = re.compile(r'private readonly MapField<(\w+), ([\w.]+)> (\w+)_;')

# private readonly RepeatedField<TreasureBoxConfigure> datas_; // 0x18
re_list_type = re.compile(r'private readonly RepeatedField<([\w.]+)> (\w+)_;')

# private string english_; // 0x38
re_other_type = re.compile(r'private ([\w.]+) (\w+)_;')

collected_types = set()
collected_proto_names = {'AppGetResponse', 'UserPostRequest'}
collected_enum_names = set()


def get_class(class_name: str) -> list[str]:
    """Read total class from dump.cs"""
    result = []
    lineno = 0
    
    # 支持多种类定义格式
    class_patterns = [
        # f'{class_name} // TypeDefIndex'.replace('_', '.'),
        # f'public class {class_name} :',
        f'public sealed class {class_name} : IMessage<{class_name}>, IMessage, IEquatable<{class_name}>, IDeepCloneable<{class_name}>, IBufferMessage // TypeDefIndex'.replace('_', '.')
    ]
    # public sealed class UIMiniMapDatas : IMessage<UIMiniMapDatas>, IMessage, IEquatable<UIMiniMapDatas>, IDeepCloneable<UIMiniMapDatas>, IBufferMessage // TypeDefIndex: 11253

    # find class
    while lineno < len(dump_cs_lines):
        if any(pattern in dump_cs_lines[lineno] for pattern in class_patterns):
            break
        lineno += 1
    else:
        raise IndexError(f"Class {class_name} not found")

    # record class with brace counting
    brace_count = 0
    for i in range(lineno, len(dump_cs_lines)):
        line = dump_cs_lines[i]
        result.append(line)
        
        brace_count += line.count('{') - line.count('}')
        if brace_count <= 0 and line == '}':
            break

    return result

def get_enum(enum_name: str) -> list[str]:
    """Read total enum from dump.cs"""
    result = []
    lineno = 0
    
    # 支持多种类定义格式
    enum_patterns = [
        f'public enum {enum_name} // TypeDefIndex'.replace('_', '.'),
    ]

    # find enum
    while lineno < len(dump_cs_lines):
        if any(pattern in dump_cs_lines[lineno] for pattern in enum_patterns):
            break
        lineno += 1
    else:
        raise IndexError(f"Enum {enum_name} not found")

    # record enum with brace counting
    brace_count = 0
    for i in range(lineno, len(dump_cs_lines)):
        line = dump_cs_lines[i]
        result.append(line)
        
        brace_count += line.count('{') - line.count('}')
        if brace_count <= 0 and line == '}':
            break

    return result


class ProtoDumper(object):
    def __init__(self, proto_name: str):
        self.proto_name = proto_name
        self.headers = ['syntax = "proto3";\n']
        self.lines = []
        self.imports = set()

        if proto_name in collected_enum_names:
            self.collect_enum(proto_name)
        else:
            self.collect_proto(proto_name)
       

    def collect_enum(self, enum_name: str) -> None:
        collected_types.add(enum_name)
        proto_declaration = f'enum {enum_name} {{\n'

        enum_class = get_enum(enum_name)

        for lineno, line in enumerate(enum_class):
            originalName = re.search(r'[OriginalName((\w+))]', line)
            if originalName == None:
                continue
            member_name = originalName.group(1)

            next_lineno = lineno + 1
            next_line = enum_class[next_lineno]
            member = re.search(r'public const (\w+) = (\w+);', next_line)
            if member == None:
                raise IndexError(f"Enum OriginalName {member_name} not found")
            member_id = member.group(2)
            proto_declaration += f'  {member_name} = {member_id};\n'

        proto_declaration += '}\n'
        self.lines.append(proto_declaration)


    def collect_proto(self, proto_name: str) -> None:
        collected_types.add(proto_name)
        proto_declaration = f'message {proto_name} {{\n'
        
        clazz = get_class(proto_name)

        for lineno, line in enumerate(clazz):
            lds = re.search(r'public const int (\w+)FieldNumber = (\w+);', line)
            if lds == None:
                continue
            

            # 查找字段定义行
            next_lineno = lineno + 1
            while next_lineno < len(clazz) and clazz[next_lineno].startswith('private static readonly'):
                next_lineno += 1
            
            if next_lineno >= len(clazz):
                continue
                
            next_line = clazz[next_lineno]
                
            member_id = lds.group(2)

            if re_dict_type.search(next_line):
                proto_declaration += self.handle_dictionary_type(next_line, member_id)
            elif re_list_type.search(next_line):
                proto_declaration += self.handle_list_type(next_line, member_id)
            else:
                proto_declaration += self.handle_type(next_line, member_id)

        proto_declaration += '}\n'
        self.lines.append(proto_declaration)

    def collect_type_if_needed(self, type_name: str) -> None:
        if type_name not in type_map and type_name not in self.imports:
            if type_name not in collected_types:
                try:
                    ProtoDumper(type_name).dump()
                except Exception as e:
                    print(f'Warning: Failed to collect type {type_name}: {e}')
            self.imports.add(type_name)

    def handle_dictionary_type(self, line: str, member_id: str) -> str:
        match_dict = re_dict_type.search(line)
        if not match_dict:
            return ""
            
        member_type_key = match_dict.group(1)
        member_type_value = match_dict.group(2)
        member_name = match_dict.group(3)

        self.collect_type_if_needed(member_type_key)
        self.collect_type_if_needed(member_type_value)

        member_type_key = type_map.get(member_type_key, member_type_key)
        member_type_value = type_map.get(member_type_value, member_type_value)

        return f'  map<{member_type_key}, {member_type_value}> {member_name} = {member_id};\n'

    def handle_list_type(self, line: str, member_id: str) -> str:
        match_list = re_list_type.search(line)
        if not match_list:
            return ""
            
        member_type_item = match_list.group(1)
        member_name = match_list.group(2)

        self.collect_type_if_needed(member_type_item)
        member_type_item = type_map.get(member_type_item, member_type_item)

        return f'  repeated {member_type_item} {member_name} = {member_id};\n'

    def handle_type(self, line: str, member_id: str) -> str:
        match_member = re_other_type.search(line)
        if not match_member:
            return ""
            
        member_type_cs = match_member.group(1)
        member_name = match_member.group(2)

        self.collect_type_if_needed(member_type_cs)
        member_type_cs = type_map.get(member_type_cs, member_type_cs)

        return f'  {member_type_cs} {member_name} = {member_id};\n'

    def dump(self) -> None:
        if not self.lines:
            return
            
        for proto_name in sorted(self.imports):
            self.headers.append(f'import "dump/{proto_name}.proto";\n')

        os.makedirs('dump', exist_ok=True)
        with open(f'dump/{self.proto_name}.proto', 'w', encoding='utf-8') as f:
            f.writelines(self.headers + self.lines)


def dump_all():
    # 只匹配 IMessage<T, T> 类型的类
    for line in dump_cs_lines:
        match = re.search(r'public sealed class (\w+) : IMessage<(\w+)', line)
        if match:
            class_name = match.group(1)
            collected_proto_names.add(class_name)
            print(f'Found IMessage class: {class_name}')

        enummatch = re.search(r'public enum (\w+) // TypeDefIndex:', line)
        if enummatch:
            enum_name = enummatch.group(1)
            collected_enum_names.add(enum_name)
            print(f'Found enum class: {enum_name}')

    # 移除不需要的类型
    collected_proto_names.discard('EmptyRequest')
    collected_proto_names.discard('EmptyResponse')
    print(f'Collected {len(collected_proto_names)} proto classes')

    for proto_name in collected_proto_names:
        try:
            ProtoDumper(proto_name).dump()
            print(f'Successfully dumped {proto_name}')
        except Exception as e:
            print(f'Error on {proto_name}: {e}')

    for enum_name in collected_enum_names:
        try:
            ProtoDumper(enum_name).dump()
            print(f'Successfully dumped enum {enum_name}')
        except Exception as e:
            print(f'Error on enum {enum_name}: {e}')


if __name__ == '__main__':
    dump_all()