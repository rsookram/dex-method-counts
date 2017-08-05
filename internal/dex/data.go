/*
Copyright 2017 Rashad Sookram
Copyright (C) 2009 The Android Open Source Project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dex

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

const (
	endianConstant        = 0x12345678
	reverseEndianConstant = 0x78563412
)

// Data extracted from a DEX file.
type Data struct {
	dexFile    os.File
	headerItem headerItem
	strings    []string
	typeIds    []typeIdItem
	protoIds   []protoIdItem
	fieldIds   []fieldIdItem
	methodIds  []methodIdItem
	classDefs  []classDefItem

	tmpBuf       []byte
	tmpSingleBuf []byte
	tmpShortBuf  []byte
	isBigEndian  bool
}

func New(f os.File) (*Data, error) {
	data := Data{
		dexFile:      f,
		tmpBuf:       make([]byte, 4),
		tmpSingleBuf: make([]byte, 1),
		tmpShortBuf:  make([]byte, 2),
	}

	err := data.load()
	if err != nil {
		return nil, err
	}

	return &data, nil
}

// Loads the contents of the DEX file into our data structures.
func (d *Data) load() error {
	err := d.parseHeaderItem()
	if err != nil {
		return err
	}

	if err = d.loadStrings(); err != nil {
		return err
	}
	if err = d.loadTypeIds(); err != nil {
		return err
	}
	if err = d.loadProtoIds(); err != nil {
		return err
	}
	if err = d.loadFieldIds(); err != nil {
		return err
	}
	if err = d.loadMethodIds(); err != nil {
		return err
	}
	if err = d.loadClassDefs(); err != nil {
		return err
	}

	d.markInternalClasses()

	return nil
}

// Parses the interesting bits out of the header.
func (d *Data) parseHeaderItem() error {
	d.headerItem = headerItem{}

	_, err := d.dexFile.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}

	magic := make([]byte, 8)
	err = readFully(d.dexFile, magic)
	if err != nil {
		return err
	}

	if !verifyMagic(magic) {
		fmt.Fprintln(os.Stderr, "Magic number is wrong -- are you sure this is a DEX file?")
		return errors.New("wrong magic number")
	}

	// Read the endian tag, so we properly swap things as we read them from here
	// on.
	_, err = d.dexFile.Seek(8+4+20+4+4, os.SEEK_SET)
	if err != nil {
		return err
	}

	endianTag, err := d.readInt()
	if err != nil {
		return err
	}

	d.headerItem.endianTag = endianTag
	if endianTag == endianConstant {
		// standard dex files are little endian
	} else if endianTag == reverseEndianConstant {
		// file is big-endian, reverse future reads
		d.isBigEndian = true
	} else {
		endianHex := fmt.Sprintf("%x", endianTag)
		fmt.Fprintln(os.Stderr, "Endian constant has unexpected value "+endianHex)
		return errors.New("unexpected endian constant " + endianHex)
	}

	// magic, checksum, signature
	_, err = d.dexFile.Seek(8+4+20, os.SEEK_SET)
	if err != nil {
		return err
	}

	fileSize, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.fileSize = fileSize

	headerSize, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.headerSize = headerSize

	// endianTag
	if _, err = d.readInt(); err != nil {
		return err
	}

	// linkSize
	if _, err = d.readInt(); err != nil {
		return err
	}

	// linkOff
	if _, err = d.readInt(); err != nil {
		return err
	}

	// mapOff
	if _, err = d.readInt(); err != nil {
		return err
	}

	stringIdsSize, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.stringIdsSize = stringIdsSize

	stringIdsOff, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.stringIdsOff = stringIdsOff

	typeIdsSize, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.typeIdsSize = typeIdsSize

	typeIdsOff, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.typeIdsOff = typeIdsOff

	protoIdsSize, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.protoIdsSize = protoIdsSize

	protoIdsOff, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.protoIdsOff = protoIdsOff

	fieldIdsSize, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.fieldIdsSize = fieldIdsSize

	fieldIdsOff, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.fieldIdsOff = fieldIdsOff

	methodIdsSize, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.methodIdsSize = methodIdsSize

	methodIdsOff, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.methodIdsOff = methodIdsOff

	classDefsSize, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.classDefsSize = classDefsSize

	classDefsOff, err := d.readInt()
	if err != nil {
		return err
	}
	d.headerItem.classDefsOff = classDefsOff

	// dataSize
	if _, err = d.readInt(); err != nil {
		return err
	}

	// dataOff
	if _, err = d.readInt(); err != nil {
		return err
	}

	return nil
}

// Loads the string table out of the DEX.
//
// First we read all of the string_id_items, then we read all of the
// string_data_item. Doing it this way should allow us to avoid seeking around
// in the file.
func (d *Data) loadStrings() error {
	count := d.headerItem.stringIdsSize
	stringOffsets := make([]int, count)

	_, err := d.dexFile.Seek(int64(d.headerItem.stringIdsOff), os.SEEK_SET)
	if err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		offset, err := d.readInt()
		if err != nil {
			return err
		}

		stringOffsets[i] = offset
	}

	d.strings = make([]string, count)

	_, err = d.dexFile.Seek(int64(stringOffsets[0]), os.SEEK_SET)
	if err != nil {
		return err
	}

	for i, offset := range stringOffsets {
		// should be a no-op
		_, err := d.dexFile.Seek(int64(offset), os.SEEK_SET)
		if err != nil {
			return err
		}

		d.strings[i], err = readString(d.dexFile, d.tmpSingleBuf)
		if err != nil {
			return err
		}
	}

	return nil
}

// Loads the type ID list.
func (d *Data) loadTypeIds() error {
	count := d.headerItem.typeIdsSize
	d.typeIds = make([]typeIdItem, count)

	if _, err := d.dexFile.Seek(int64(d.headerItem.typeIdsOff), os.SEEK_SET); err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		d.typeIds[i] = typeIdItem{}

		descriptorIdx, err := d.readInt()
		if err != nil {
			return err
		}

		d.typeIds[i].descriptorIdx = descriptorIdx
	}

	return nil
}

// Loads the proto ID list.
func (d *Data) loadProtoIds() error {
	count := d.headerItem.protoIdsSize
	d.protoIds = make([]protoIdItem, count)

	if _, err := d.dexFile.Seek(int64(d.headerItem.protoIdsOff), os.SEEK_SET); err != nil {
		return err
	}

	// Read the proto ID items.
	for i := 0; i < count; i++ {
		d.protoIds[i] = protoIdItem{}

		shortyIdx, err := d.readInt()
		if err != nil {
			return err
		}

		d.protoIds[i].shortyIdx = shortyIdx

		returnTypeIdx, err := d.readInt()
		if err != nil {
			return err
		}

		d.protoIds[i].returnTypeIdx = returnTypeIdx

		parametersOff, err := d.readInt()
		if err != nil {
			return err
		}

		d.protoIds[i].parametersOff = parametersOff
	}

	// Go back through and read the type lists.
	for _, protoId := range d.protoIds {
		offset := protoId.parametersOff

		if offset == 0 {
			protoId.types = make([]int16, 0)
			continue
		}

		if _, err := d.dexFile.Seek(int64(offset), os.SEEK_SET); err != nil {
			return err
		}

		size, err := d.readInt() // #of entries in list
		if err != nil {
			return err
		}

		protoId.types = make([]int16, size)

		for j := 0; j < size; j++ {
			t, err := d.readShort()
			if err != nil {
				return err
			}

			protoId.types[j] = t
		}
	}

	return nil
}

// Loads the field ID list.
func (d *Data) loadFieldIds() error {
	count := d.headerItem.fieldIdsSize
	d.fieldIds = make([]fieldIdItem, count)

	if _, err := d.dexFile.Seek(int64(d.headerItem.fieldIdsOff), os.SEEK_SET); err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		d.fieldIds[i] = fieldIdItem{}

		classIdx, err := d.readShort()
		if err != nil {
			return err
		}

		d.fieldIds[i].classIdx = classIdx

		typeIdx, err := d.readShort()
		if err != nil {
			return err
		}

		d.fieldIds[i].typeIdx = typeIdx

		nameIdx, err := d.readInt()
		if err != nil {
			return err
		}

		d.fieldIds[i].nameIdx = nameIdx
	}

	return nil
}

// Loads the method ID list.
func (d *Data) loadMethodIds() error {
	count := d.headerItem.methodIdsSize
	d.methodIds = make([]methodIdItem, count)

	if _, err := d.dexFile.Seek(int64(d.headerItem.methodIdsOff), os.SEEK_SET); err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		d.methodIds[i] = methodIdItem{}

		classIdx, err := d.readShort()
		if err != nil {
			return err
		}

		d.methodIds[i].classIdx = classIdx

		protoIdx, err := d.readShort()
		if err != nil {
			return err
		}

		d.methodIds[i].protoIdx = protoIdx

		nameIdx, err := d.readInt()
		if err != nil {
			return err
		}

		d.methodIds[i].nameIdx = nameIdx
	}

	return nil
}

// Loads the class defs list
func (d *Data) loadClassDefs() error {
	count := d.headerItem.classDefsSize
	d.classDefs = make([]classDefItem, count)

	if _, err := d.dexFile.Seek(int64(d.headerItem.classDefsOff), os.SEEK_SET); err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		d.classDefs[i] = classDefItem{}

		classIdx, err := d.readInt()
		if err != nil {
			return err
		}

		d.classDefs[i].classIdx = classIdx

		// access_flags
		if _, err = d.readInt(); err != nil {
			return err
		}
		// superclass_idx
		if _, err = d.readInt(); err != nil {
			return err
		}
		// interfaces_off
		if _, err = d.readInt(); err != nil {
			return err
		}
		// source_file_idx
		if _, err = d.readInt(); err != nil {
			return err
		}
		// annotations_off
		if _, err = d.readInt(); err != nil {
			return err
		}
		// class_data_off
		if _, err = d.readInt(); err != nil {
			return err
		}
		// static_values_off
		if _, err = d.readInt(); err != nil {
			return err
		}
	}

	return nil
}

// Sets the "internal" flag on type IDs which are defined in the DEX file or
// within  the VM (e.g. primitive classes and arrays).
func (d *Data) markInternalClasses() {
	for _, classDef := range d.classDefs {
		d.typeIds[classDef.classIdx].internal = true
	}

	for _, typeId := range d.typeIds {
		className := d.strings[typeId.descriptorIdx]

		if len(className) == 1 {
			// primitive class
			typeId.internal = true
		} else if className[0] == '[' {
			typeId.internal = true
		}
	}
}

// Verifies the given magic number.
func verifyMagic(magic []byte) bool {
	dexFileMagic := []byte{0x64, 0x65, 0x78, 0x0a, 0x30, 0x33, 0x36, 0x00}
	dexFileMagicApi13 := []byte{0x64, 0x65, 0x78, 0x0a, 0x30, 0x33, 0x35, 0x00}
	return bytes.Equal(magic, dexFileMagic) || bytes.Equal(magic, dexFileMagicApi13)
}

// Queries

func (d *Data) classNameFromTypeIndex(idx int16) string {
	return d.strings[d.typeIds[idx].descriptorIdx]
}

func (d *Data) argArrayFromProtoIndex(idx int16) []string {
	protoId := d.protoIds[idx]

	result := make([]string, len(protoId.types))

	for i, ty := range protoId.types {
		result[i] = d.strings[d.typeIds[ty].descriptorIdx]
	}

	return result
}

func (d *Data) returnTypeFromProtoIndex(idx int16) string {
	protoId := d.protoIds[idx]
	return d.strings[d.typeIds[protoId.returnTypeIdx].descriptorIdx]
}

// Returns a slice with all of the class references that don't correspond to
// classes in the DEX file.  Each class reference has a list of the referenced
// fields and methods associated with that class.
func (d *Data) GetExternalReferences() []ClassRef {
	// create a sparse array of ClassRef that parallels d.typeIds
	sparseRefs := make([]ClassRef, len(d.typeIds))

	// create entries for all externally-referenced classes
	count := 0
	for i, typeId := range d.typeIds {
		if !typeId.internal {
			sparseRefs[i] = ClassRef{ClassName: d.strings[typeId.descriptorIdx]}
			count++
		}
	}

	// add fields and methods to the appropriate class entry
	d.addExternalFieldReferences(sparseRefs)
	d.addExternalMethodReferences(sparseRefs)

	// crunch out the sparseness
	classRefs := make([]ClassRef, count)
	idx := 0
	for i, _ := range d.typeIds {
		ref := sparseRefs[i]
		if ref.ClassName != "" || len(ref.FieldRefs) != 0 || len(ref.MethodRefs) != 0 {
			classRefs[idx] = sparseRefs[i]
			idx++
		}
	}

	return classRefs
}

// Runs through the list of field references, inserting external references
// into the appropriate ClassRef.
func (d *Data) addExternalFieldReferences(sparseRefs []ClassRef) {
	for _, fieldId := range d.fieldIds {
		if !d.typeIds[fieldId.classIdx].internal {
			newFieldRef := FieldRef{
				DeclClass: d.classNameFromTypeIndex(fieldId.classIdx),
				FieldType: d.classNameFromTypeIndex(fieldId.typeIdx),
				FieldName: d.strings[fieldId.nameIdx],
			}

			classRef := sparseRefs[fieldId.classIdx]
			classRef.FieldRefs = append(classRef.FieldRefs, newFieldRef)
		}
	}
}

// Runs through the list of method references, inserting external references
// into the appropriate ClassRef.
func (d *Data) addExternalMethodReferences(sparseRefs []ClassRef) {
	for _, methodId := range d.methodIds {
		if !d.typeIds[methodId.classIdx].internal {
			newMethodRef := MethodRef{
				DeclClass:  d.classNameFromTypeIndex(methodId.classIdx),
				ArgTypes:   d.argArrayFromProtoIndex(methodId.protoIdx),
				ReturnType: d.returnTypeFromProtoIndex(methodId.protoIdx),
				MethodName: d.strings[methodId.nameIdx],
			}

			classRef := sparseRefs[methodId.classIdx]
			classRef.MethodRefs = append(classRef.MethodRefs, newMethodRef)
		}
	}
}

func (d *Data) GetMethodRefs() []MethodRef {
	methodRefs := make([]MethodRef, len(d.methodIds))
	for i, methodId := range d.methodIds {
		methodRefs[i] = MethodRef{
			DeclClass:  d.classNameFromTypeIndex(methodId.classIdx),
			ArgTypes:   d.argArrayFromProtoIndex(methodId.protoIdx),
			ReturnType: d.returnTypeFromProtoIndex(methodId.protoIdx),
			MethodName: d.strings[methodId.nameIdx],
		}
	}
	return methodRefs
}

func (d *Data) GetFieldRefs() []FieldRef {
	fieldRefs := make([]FieldRef, len(d.fieldIds))
	for i, fieldId := range d.fieldIds {
		fieldRefs[i] = FieldRef{
			DeclClass: d.classNameFromTypeIndex(fieldId.classIdx),
			FieldType: d.classNameFromTypeIndex(fieldId.typeIdx),
			FieldName: d.strings[fieldId.nameIdx],
		}
	}
	return fieldRefs
}

// Basic I/O Functions
func readFully(f os.File, buf []byte) error {
	n, err := f.Read(buf)
	if err != nil {
		return err
	}

	if n != len(buf) {
		return errors.New("expected to read " + string(len(buf)) + " bytes, but only read " + string(n))
	}

	return nil
}

func readByte(f os.File, buf []byte) (byte, error) {
	n, err := f.Read(buf)
	if err != nil {
		return 0, err
	}

	if n != len(buf) {
		err = errors.New("expected to read " + string(len(buf)) + " bytes, but only read " + string(n))
		return 0, err
	}

	return buf[0], nil
}

// Reads a variable-length unsigned LEB128 value. Does not attempt to verify
// that the value is valid.
func readUnsignedLeb128(f os.File, buf []byte) (uint32, error) {
	result := uint32(0)
	var val byte = 0x80
	var err error

	// Stop when the highest bit is clear
	for val >= 0x80 {
		val, err = readByte(f, buf)
		if err != nil {
			return 0, err
		}

		result = (result << 7) | uint32(val&0x7f)
	}

	return result, nil
}

// Reads a UTF-8 string.
//
// We don't know how long the UTF-8 string is, so we have to read one byte at a
// time.  We could make an educated guess based on the utf16_size and seek back
// if we get it wrong, but seeking backward may cause the underlying
// implementation to reload I/O buffers.
func readString(f os.File, buf []byte) (string, error) {
	utf16len, err := readUnsignedLeb128(f, buf)
	//fmt.Println(utf16len)
	if err != nil {
		return "", err
	}

	inBuf := make([]byte, utf16len*3) // worst case
	var idx int

	for idx = 0; idx < len(inBuf); idx++ {
		val, err := readByte(f, buf)
		if err != nil {
			return "", err
		}

		if val == 0x00 {
			break
		}
		inBuf[idx] = val
	}

	return string(inBuf[:idx]), nil
}

// Reads a signed 16-bit integer, byte-swapping if necessary
func (d *Data) readShort() (int16, error) {
	if err := readFully(d.dexFile, d.tmpShortBuf); err != nil {
		return -1, err
	}

	buf := bytes.NewBuffer(d.tmpShortBuf)
	var ret int16
	if d.isBigEndian {
		binary.Read(buf, binary.BigEndian, &ret)
	} else {
		binary.Read(buf, binary.LittleEndian, &ret)
	}

	return ret, nil
}

// Reads a signed 32-bit integer, byte-swapping if necessary
func (d *Data) readInt() (int, error) {
	err := readFully(d.dexFile, d.tmpBuf)
	if err != nil {
		return -1, err
	}

	buf := bytes.NewBuffer(d.tmpBuf)
	var ret int32
	if d.isBigEndian {
		binary.Read(buf, binary.BigEndian, &ret)
	} else {
		binary.Read(buf, binary.LittleEndian, &ret)
	}

	return int(ret), nil
}

// Internal "structure" declarations

// Holds the contents of a header_item.
type headerItem struct {
	fileSize      int
	headerSize    int
	endianTag     int
	stringIdsSize int
	stringIdsOff  int
	typeIdsSize   int
	typeIdsOff    int
	protoIdsSize  int
	protoIdsOff   int
	fieldIdsSize  int
	fieldIdsOff   int
	methodIdsSize int
	methodIdsOff  int
	classDefsSize int
	classDefsOff  int
}

// Holds the contents of a type_id_item.
//
// This is chiefly a list of indices into the string table.  We need some
// additional bits of data, such as whether or not the type ID represents a
// class defined in this DEX, so we use a struct for each instead of a simple
// integer.
type typeIdItem struct {
	descriptorIdx int  // index into string_ids
	internal      bool // defined within this DEX file?
}

// Holds the contents of a proto_id_item.
type protoIdItem struct {
	shortyIdx     int // index into string_ids
	returnTypeIdx int // index into type_ids
	parametersOff int // file offset to a type_list

	types []int16 // contents of type list
}

// Holds the contents of a field_id_item.
type fieldIdItem struct {
	classIdx int16 // index into type_ids (defining class)
	typeIdx  int16 // index into type_ids (field type)
	nameIdx  int   // index into string_ids
}

// Holds the contents of a method_id_item.
type methodIdItem struct {
	classIdx int16 // index into type_ids
	protoIdx int16 // index into proto_ids
	nameIdx  int   // index into string_ids
}

// Holds the contents of a class_def_item.
//
// We don't really need a class for this, but there's some stuff in the
// class_def_item that we might want later.
type classDefItem struct {
	classIdx int // index into type_ids
}
