# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# NO CHECKED-IN PROTOBUF GENCODE
# source: encoder/encoder.proto
# Protobuf Python Version: 5.27.1
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(
    _runtime_version.Domain.PUBLIC,
    5,
    27,
    1,
    '',
    'encoder/encoder.proto'
)
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


from validate import validate_pb2 as validate_dot_validate__pb2


DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x15\x65ncoder/encoder.proto\x1a\x17validate/validate.proto\"~\n\x07\x45ncoder\x12Q\n\x14sentence_transformer\x18\x03 \x01(\x0b\x32\x1c.Encoder.SentenceTransformerH\x00R\x13sentenceTransformer\x1a\x15\n\x13SentenceTransformerB\t\n\x07\x65ncoderB.B\x0c\x45ncoderProtoP\x01Z\x1c\x62igkinds.or.kr/proto/encoderb\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'encoder.encoder_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
  _globals['DESCRIPTOR']._loaded_options = None
  _globals['DESCRIPTOR']._serialized_options = b'B\014EncoderProtoP\001Z\034bigkinds.or.kr/proto/encoder'
  _globals['_ENCODER']._serialized_start=50
  _globals['_ENCODER']._serialized_end=176
  _globals['_ENCODER_SENTENCETRANSFORMER']._serialized_start=144
  _globals['_ENCODER_SENTENCETRANSFORMER']._serialized_end=165
# @@protoc_insertion_point(module_scope)
