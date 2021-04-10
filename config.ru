#!/usr/bin/env rackup
# frozen_string_literal: true

require "bundler/setup"
Bundler.require(:default)

Dir[File.join(__dir__, "lib", "*.rb")].each { |file| require file }
Dir[File.join(__dir__, "models", "*.rb")].each { |file| require file }

require "#{File.dirname(__FILE__)}/main.rb"

run Code
