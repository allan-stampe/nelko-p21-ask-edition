#!/usr/bin/env python3
"""Verify exact ICO payloads in the final PE resources. Python standard library only."""
import argparse
from pathlib import Path
import struct

p=argparse.ArgumentParser()
p.add_argument('--ico',type=Path,default=Path('cmd/nelko-print/assets/app-icon.ico'))
p.add_argument('--png',type=Path,default=Path('cmd/nelko-print/assets/app-icon.png'))
p.add_argument('--exe',type=Path)
a=p.parse_args()
ico=a.ico.read_bytes()
assert struct.unpack_from('<HHH',ico)==(0,1,4), 'ICO must have four images'
images={}
for i in range(4):
    w,h,colors,reserved,planes,bpp,length,offset=struct.unpack_from('<BBBBHHII',ico,6+16*i)
    size=(w or 256,h or 256)
    payload=ico[offset:offset+length]
    assert len(payload)==length
    assert payload[:8]==b'\x89PNG\r\n\x1a\n'
    assert struct.unpack_from('>II',payload,16)==size
    assert size not in images
    images[size]=payload
assert set(images)=={(16,16),(32,32),(48,48),(256,256)}
assert images[(256,256)]==a.png.read_bytes(), 'Window and EXE icons differ'
print('ICO: exact 16/32/48/256 images; embedded window PNG matches 256 image.')
if not a.exe:
    raise SystemExit(0)
b=a.exe.read_bytes()
assert b[:2]==b'MZ'
pe=struct.unpack_from('<I',b,0x3c)[0]
assert b[pe:pe+4]==b'PE\0\0'
machine,nsections,_,_,_,optional_size,_=struct.unpack_from('<HHIIIHH',b,pe+4)
assert machine==0x8664,'Not Windows AMD64'
opt=pe+24
assert struct.unpack_from('<H',b,opt)[0]==0x20b,'Not PE32+'
assert struct.unpack_from('<H',b,opt+68)[0]==2,'Not GUI subsystem'
resource_rva,resource_size=struct.unpack_from('<II',b,opt+112+8*2)
assert resource_rva and resource_size,'No .rsrc directory'
sections=[]
for i in range(nsections):
    s=opt+optional_size+40*i
    virtual_size,rva,raw_size,raw_offset=struct.unpack_from('<IIII',b,s+8)
    sections.append((rva,max(virtual_size,raw_size),raw_offset))
def file_offset(rva):
    for base,size,raw in sections:
        if base<=rva<base+size:return raw+rva-base
    raise AssertionError(f'Unmapped RVA {rva:x}')
base=file_offset(resource_rva)
def entries(offset):
    named,ids=struct.unpack_from('<HH',b,base+offset+12)
    return [struct.unpack_from('<II',b,base+offset+16+8*i) for i in range(named+ids)]
resources={}
def walk(offset,path=()):
    for name,target in entries(offset):
        assert name&0x80000000==0,'Unexpected named resource'
        key=path+(name,)
        if target&0x80000000:walk(target&0x7fffffff,key)
        else:
            rva,length,_,_=struct.unpack_from('<IIII',b,base+target)
            resources[key]=b[file_offset(rva):file_offset(rva)+length]
walk(0)
groups=[(key,data) for key,data in resources.items() if key[0]==14]
assert len(groups)==1,'Expected one RT_GROUP_ICON'
key,group=groups[0]
assert struct.unpack_from('<HHH',group)==(0,1,4)
found=set()
for i in range(4):
    w,h,_,_,planes,bpp,length,icon_id=struct.unpack_from('<BBBBHHIH',group,6+14*i)
    size=(w or 256,h or 256)
    assert size in images and size not in found
    payload=resources[(3,icon_id,key[-1])]
    assert payload==images[size] and len(payload)==length, f'PE icon mismatch at {size}'
    found.add(size)
assert found==set(images)
assert any(k[0]==16 for k in resources),'Missing Windows version information'
print('PE32+ AMD64 GUI: all four embedded icons exactly match the ICO; VERSIONINFO exists.')
