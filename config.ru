#!/usr/bin/env rackup
# encoding: utf-8

require 'bundler/setup'
Bundler.require(:default)

Dir[File.join(__dir__, 'lib', '*.rb')].each { |file| require file }
Dir[File.join(__dir__, 'models', '*.rb')].each { |file| require file }

require File.dirname(__FILE__) + "/database.rb"
require File.dirname(__FILE__) + "/main.rb"

run Code
