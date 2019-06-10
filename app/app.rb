class Code < Padrino::Application
  register SassInitializer
  use ActiveRecord::ConnectionAdapters::ConnectionManagement
  register Padrino::Rendering
  register Padrino::Helpers

  enable :sessions

  register Padrino::Cache
  enable :caching
  if Padrino.env.eql? :production && ENV["MEMCACHIER_SERVERS"] && ENV["MEMCACHIER_USERNAME"] && ENV["MEMCACHIER_PASSWORD"]
cache = Dalli::Client.new((ENV["MEMCACHIER_SERVERS"] || "").split(","), {
      :username => ENV["MEMCACHIER_USERNAME"],
      :password => ENV["MEMCACHIER_PASSWORD"],
      :failover => true,
      :socket_timeout => 1.5,
      :socket_failure_delay => 0.2,
    })
    set :cache, Padrino::Cache.new(:Memcached, :backend => cache)
  else
    set :cache, Padrino::Cache.new(:File, :dir => Padrino.root('tmp', app_name.to_s, 'cache')) # default choice
  end
end
