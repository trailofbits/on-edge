#======================================================================================================#
# Copyright 2019 Trail of Bits. All rights reserved.
# Makefile for OnEdge
#======================================================================================================#

TEST_FLAGS := -test.failfast -test.v

.PHONY: test basic_test nested_test onedge.test vet

test: basic_test nested_test

basic_test: onedge.test
	./$< $(TEST_FLAGS) -test.run TestBasic

nested_test: onedge.test
	./$< $(TEST_FLAGS) -test.run TestNested

onedge.test:
	go test -race -c

vet:
	go vet -race -tests=false

#======================================================================================================#
