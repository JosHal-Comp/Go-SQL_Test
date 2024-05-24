CREATE TABLE transport_data (
	area1	varchar(255),
	prov_id1	int,
	city_id1	int,
	dist_id1	int,
	area_id1	varchar(255),
	area2	varchar(255),
	prov_id2	int,
	city_id2	int,
	dist_id2	int,
	area_id2	varchar(255)
);

COPY transport_data (area1, area2) FROM 'FILEPATH';

CREATE TABLE area_info (
	province	varchar(255),
	prov_id		int,
	city		varchar(255),
	city_id		int,
	district	varchar(255),
	dist_id		int
);

-- area_info created by joining three tables together (province, city and district) --