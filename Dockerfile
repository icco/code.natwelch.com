From ruby:3.0.0

WORKDIR /opt
COPY . .

ENV PORT 8080
ENV RACK_ENV production

RUN bundle config set --local system 'true'
RUN bundle config set --local without 'test development'
RUN bundle install

CMD bundle exec thin -R config.ru start -p $PORT
EXPOSE 8080
