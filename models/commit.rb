class Commit <  ActiveRecord::Base
  USER = "icco"

  validates :user, :presence => true
  validates :repo, :presence => true
  validates :sha, :presence => true, :uniqueness => {:scope => [:repo]}

  def to_s
    "#{user}/#{repo}##{sha}"
  end

  # Grabs the commit log from github archive for the specified hour, parses
  # that and then saves all commits pushed by the USER to the database.
  def self.fetchAllForTime day, month, year, hour, client = nil
    require "open-uri"
    require "zlib"

    # Simple error checking.
    return nil if hour < 0 or hour > 23
    return nil if day < 1 or day > 31
    return nil if month < 1 or month > 12

    date = "#{"%4d" % year}-#{"%02d" % month}-#{"%02d" % day}-#{hour}"
    uri = URI.parse "http://data.githubarchive.org/#{date}.json.gz"
    parser = Yajl::Parser.new(:symbolize_keys => true)

    logger.info "Grabbing #{date}"

    begin
      uri.open do |gz|
        js = Zlib::GzipReader.new(gz).read
        parser.parse(js) do |event|
          if event[:actor] == USER and event[:type] == "PushEvent"
            # TODO(icco): fix this so we record the user name of the commit
            # owner, not the repo owner.
            user = event[:repository][:owner]
            repo = event[:repository][:name]
            event[:payload][:shas].each do |commit|
              sha = commit[0]
              ret = self.factory user, repo, sha, client
              if !ret.nil?
                logger.info "Inserted #{ret}."
              end
            end
          end
        end
      end
    rescue Timeout::Error
      logger.push "The request for #{uri} timed out...skipping.", :warn
    rescue OpenURI::HTTPError => e
      logger.push "The request for #{uri} returned an error. #{e.message}", :warn
    end
  end

  # This creates a Commit.
  #
  # NOTE: repo + sha are supposed to be unique, so if those two already exist,
  # but the user is wrong, we'll try and update (if check? is true).
  def self.factory user, repo, sha, client = nil, check = false
    if client.nil?
      client = Octokit::Client.new({})
    end

    commit = Commit.where(:repo => repo, :sha => sha).first_or_initialize
    if !commit.new_record? and !commit.changed? and !check
      logger.push "#{user}/#{repo}##{sha} already exists as #{commit.inspect}.", :info
      # We return nil for better logging above.
      return nil
    end

    # No need to check.
    if check and !commit.new_record? and commit.user.eql? user
      return nil
    end

    raise "Github ratelimit remaining #{client.ratelimit.remaining} of #{client.ratelimit.limit} is not enough." if client.ratelimit.remaining <= 2

    begin
      gh_commit = client.commit("#{user}/#{repo}", sha)

      # This is to prevent counting repos I just forked and didn't do any work
      # in. A few commits will still slip through thought that don't belong to
      # me. I don't know why.
      if gh_commit.author and gh_commit.author.login
        commit.user = gh_commit.author.login
      else
        commit.user = user
      end

      commit.repo = repo
      commit.sha = sha

      create_date = gh_commit.commit.author.date
      if create_date.is_a? String
        commit.created_on = DateTime.iso8601(create_date)
      else
        commit.created_on = create_date
      end

      if commit.valid?
        commit.save
        return c
      else
        logger.push("Error Saving Commit #{user}/#{repo}:#{commit.sha}: #{commit.errors.messages.inspect}", :warn)
        return nil
      end
    rescue Octokit::NotFound
      logger.push("Error Saving Commit #{user}/#{repo}:#{sha}: 404", :warn)
    end
  end
end
