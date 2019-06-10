module SassInitializer
  def self.registered(app)
    # Enables support for SASS template reloading in rack applications.
    # See http://nex-3.com/posts/88-sass-supports-rack for more details.
    # Store SASS files (by default) within 'app/stylesheets'.
    require 'rack/sassc'

    app.use Rack::SassC, {
      always_update: (Padrino.env == :development),
      css_location: Padrino.root("public/css"),
      full_exception: (Padrino.env == :development),
      never_update: (Padrino.env == :production),
      style: :compact,
      template_location: Padrino.root("app/css"),
    }
  end
end
