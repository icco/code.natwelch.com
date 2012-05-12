DB = Sequel.connect(ENV['DATABASE_URL'] || 'sqlite://db/data.db')

class Repo < Sequel::Model(:repos)
end
