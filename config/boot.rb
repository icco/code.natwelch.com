# Defines our constants
RACK_ENV = ENV['RACK_ENV'] ||= 'development' unless defined?(RACK_ENV)
PADRINO_ROOT = File.expand_path('../..', __FILE__) unless defined?(PADRINO_ROOT)

# Load our dependencies
require 'rubygems' unless defined?(Gem)
require 'bundler/setup'
Bundler.require(:default, RACK_ENV)

##
## Enable devel logging
Padrino::Logger::Config[:development][:log_level]  = :devel
Padrino::Logger::Config[:development][:log_static] = true

Padrino::Logger::Config[:production][:log_level]  = :info
Padrino::Logger::Config[:production][:stream] = :stdout

## Configure your I18n
I18n.default_locale = :en

##
# Add your before (RE)load hooks here
#
Padrino.before_load do
end

##
# Add your after (RE)load hooks here
#
Padrino.after_load do
  logger.info "Running as #{Padrino.env.inspect}."
  logger.info "Logger: #{logger.inspect}"
end

Padrino.load!
