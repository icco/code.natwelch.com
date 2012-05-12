DB = Sequel.connect(ENV['DATABASE_URL'] || 'sqlite://db/data.db')

class Repo < Sequel::Model(:repos)
  def self.factory name, user, watchers, forks
    r = Repo.new
    r.user = user
    r.repo = name
    r.watchers = watchers
    r.forks = forks
    r.created_on = Time.now
    r.save

    return r
  end
end

class StatEntry < Sequel::Model(:entries)
end