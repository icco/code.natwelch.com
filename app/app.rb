class Code < Padrino::Application
  register LessInitializer
  use ActiveRecord::ConnectionAdapters::ConnectionManagement
  register Padrino::Rendering
  register Padrino::Helpers

  enable :sessions

  register Padrino::Cache
  enable :caching
end
