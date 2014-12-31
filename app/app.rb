class Code < Padrino::Application
  register SassInitializer
  use ActiveRecord::ConnectionAdapters::ConnectionManagement
  register Padrino::Rendering
  register Padrino::Helpers

  enable :sessions

  register Padrino::Cache
  enable :caching
  if Padrino.env.eql? :production
    auth_pair = [
      ENV['MEMCACHIER_PASSWORD'],
      ENV['MEMCACHIER_USERNAME']
    ]
    set :cache, Padrino::Cache.new(:Memcached, ENV['MEMCACHIER_SERVERS'], :credentials => auth_pair, :exception_retry_limit => 1)
  else
    set :cache, Padrino::Cache.new(:File, :dir => Padrino.root('tmp', app_name.to_s, 'cache')) # default choice
  end
end
