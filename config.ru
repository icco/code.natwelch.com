#!/usr/bin/env rackup
# encoding: utf-8

require 'bundler/setup'
Bundler.require(:default)

require File.dirname(__FILE__) + "/lib/connection_pool_management_middleware.rb"
require File.dirname(__FILE__) + "/lib/sass_initializer.rb"
require File.dirname(__FILE__) + "/main.rb"

run Code
